package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"syscall"
	"time"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/cio"
	"github.com/containerd/containerd/namespaces"
	"github.com/containerd/containerd/oci"
)

type Info struct {
	ID        string
	Timestamp time.Time
	Data      interface{}
}

type ContainerService interface {
	Deploy(ctx context.Context, id string) error
	Undeploy(ctx context.Context, id string) error
	Info(ctx context.Context, id string) (*Info, error)
}

type CreationListener interface {
	Created(id string, pid uint32)
}

type service struct {
	getter    ConfigService
	listeners []CreationListener
}

func NewContainerService(getter ConfigService, listeners ...CreationListener) ContainerService {
	return &service{getter, listeners}
}

func (c *service) Deploy(ctx context.Context, id string) error {
	log.Printf("Conneting to containerd\n")
	client, err := containerd.New("/run/containerd/containerd.sock")
	if err != nil {
		return err
	}
	defer client.Close()

	log.Printf("Getting container configuration\n")
	config, err := c.getter.Get(id)
	if err != nil {
		return err
	}

	ctx = namespaces.WithNamespace(ctx, "default")

	log.Printf("Deployng %s...\n", config.ContainerID)
	defer log.Printf("Deploy done")

	if _, err := c.ensureTask(ctx, client, config); err != nil {
		return err
	}

	return nil
}

func (c *service) Undeploy(ctx context.Context, id string) error {
	log.Printf("Conneting to containerd\n")
	client, err := containerd.New("/run/containerd/containerd.sock")
	if err != nil {
		return err
	}
	defer client.Close()

	ctx = namespaces.WithNamespace(ctx, "default")

	container, err := client.LoadContainer(ctx, id)
	if err != nil {
		return fmt.Errorf("container %s not found: %v", id, err)
	}

	if err := ensureTaskDelete(ctx, container); err != nil {
		return err
	}

	return nil
}

func (c *service) Info(ctx context.Context, id string) (*Info, error) {
	log.Printf("Conneting to containerd\n")
	client, err := containerd.New("/run/containerd/containerd.sock")
	if err != nil {
		return nil, err
	}
	defer client.Close()

	ctx = namespaces.WithNamespace(ctx, "default")
	container, err := client.LoadContainer(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("container %s not found: %v", id, err)
	}

	task, err := container.Task(ctx, nil)
	if err != nil {
		return nil, err
	}

	metrics, err := task.Metrics(ctx)
	if err != nil {
		return nil, err
	}

	return &Info{
		ID:        metrics.ID,
		Timestamp: metrics.Timestamp,
		Data:      metrics.Data,
	}, nil
}

func (c *service) ensureTask(ctx context.Context, client *containerd.Client,
	config *Config) (task containerd.Task, err error) {

	container, err := ensureContainer(ctx, client, config)
	if err != nil {
		return nil, err
	}

	task, err = container.Task(ctx, nil)
	if err != nil {
		return c.createTask(ctx, container)
	}

	return task, nil
}

func (c *service) createTask(ctx context.Context, container containerd.Container) (_ containerd.Task, errRet error) {

	log.Printf("Creating task for container %s\n", container.ID())
	task, err := container.NewTask(ctx, cio.NewCreator(cio.WithStdio))
	if err != nil {
		return nil, err
	}
	defer func() {
		if errRet != nil {
			task.Delete(ctx)
		}
	}()

	for _, l := range c.listeners {
		l.Created(container.ID(), task.Pid())
	}

	log.Printf("Starting task for container %s\n", container.ID())
	if err := task.Start(ctx); err != nil {
		return nil, err
	}

	return task, nil
}

func ensureTaskDelete(ctx context.Context, container containerd.Container) error {

	task, err := container.Task(ctx, nil)
	if err != nil {
		if strings.Contains(err.Error(), "no running task found") {
			return nil
		}

		return err
	}

	status, err := task.Status(ctx)
	if err != nil {
		return err
	}

	defer task.Delete(ctx)

	if status.Status == containerd.Stopped {
		return nil
	}

	exitStatus, err := stopWaitTask(ctx, task)
	if err != nil {
		return err
	}

	if exitStatus.Error() != nil {
		log.Printf("Exit status of the task: %v\n", exitStatus.Error())
	}

	return nil
}

func stopWaitTask(ctx context.Context, task containerd.Task) (*containerd.ExitStatus, error) {

	exitStatusChan, err := task.Wait(ctx)
	if err != nil {
		return nil, err
	}

	if err := task.Kill(ctx, syscall.SIGTERM); err != nil {
		return nil, err
	}

	select {
	case exitStatus := <-exitStatusChan:
		return &exitStatus, nil
	case <-ctx.Done():
		return nil, fmt.Errorf("done waiting task")
	}
}

func ensureContainer(ctx context.Context, client *containerd.Client,
	config *Config) (containerd.Container, error) {

	container, err := client.LoadContainer(ctx, config.ContainerID)
	if err != nil {
		return createContainer(ctx, client, config)
	}

	image, err := container.Image(ctx)
	if err != nil {
		return nil, err
	}

	if image.Name() != config.ImageRef {
		if err := deleteContainer(ctx, container); err != nil {
			return nil, err
		}

		return createContainer(ctx, client, config)
	}

	return container, nil
}

func createContainer(ctx context.Context, client *containerd.Client,
	config *Config) (containerd.Container, error) {

	image, err := ensureImage(ctx, client, config)
	if err != nil {
		return nil, err
	}

	log.Printf("Creating container %s\n", config.ContainerID)
	return client.NewContainer(ctx, config.ContainerID,
		containerd.WithImage(image),
		containerd.WithNewSnapshot(config.ContainerID+"-snapshot", image),
		containerd.WithNewSpec(oci.WithImageConfig(image),
			oci.WithCapabilities([]string{"CAP_NET_RAW"})))
}

func deleteContainer(ctx context.Context, container containerd.Container) error {

	if err := ensureTaskDelete(ctx, container); err != nil {
		return err
	}

	return container.Delete(ctx, containerd.WithSnapshotCleanup)
}

func ensureImage(ctx context.Context, client *containerd.Client,
	config *Config) (containerd.Image, error) {

	image, err := client.GetImage(ctx, config.ImageRef)
	if err != nil {
		return pullImage(ctx, client, config.ImageRef)
	}

	return image, nil
}

func pullImage(ctx context.Context, client *containerd.Client, ref string) (containerd.Image, error) {
	log.Printf("Pulling image %s\n", ref)
	return client.Pull(ctx, ref, containerd.WithPullUnpack)
}