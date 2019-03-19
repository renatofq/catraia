package main

import (
	"context"
	"errors"
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

type ContainerdConfig struct {
	Namespace string
	Socket    string
}

type service struct {
	conf          *ContainerdConfig
	configService ImageInfoService
	listeners     []CreationListener
}

func NewContainerService(conf *ContainerdConfig, imageService ImageInfoService, listeners ...CreationListener) ContainerService {
	return &service{conf, imageService, listeners}
}

func (c *service) Deploy(ctx context.Context, id string) error {
	log.Printf("Conneting to containerd\n")
	client, err := containerd.New(c.conf.Socket)
	if err != nil {
		return err
	}
	defer client.Close()

	log.Printf("Getting image configuration\n")
	imageInfo, err := c.configService.Get(id)
	if err != nil {
		return err
	}

	if imageInfo == nil {
		return errors.New("image does not exist at image service")
	}

	ctx = namespaces.WithNamespace(ctx, c.conf.Namespace)

	log.Printf("Deployng %s...\n", imageInfo.ID)

	if _, err := c.ensureTask(ctx, client, imageInfo); err != nil {
		return err
	}

	log.Printf("Deploy done")

	return nil
}

func (c *service) Undeploy(ctx context.Context, id string) error {
	log.Printf("Conneting to containerd\n")
	client, err := containerd.New(c.conf.Socket)
	if err != nil {
		return err
	}
	defer client.Close()

	ctx = namespaces.WithNamespace(ctx, c.conf.Namespace)

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
	client, err := containerd.New(c.conf.Socket)
	if err != nil {
		return nil, err
	}
	defer client.Close()

	ctx = namespaces.WithNamespace(ctx, c.conf.Namespace)
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
	imageInfo *ImageInfo) (task containerd.Task, err error) {

	container, err := ensureContainer(ctx, client, imageInfo)
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
	imageInfo *ImageInfo) (containerd.Container, error) {

	container, err := client.LoadContainer(ctx, imageInfo.ID)
	if err != nil {
		return createContainer(ctx, client, imageInfo)
	}

	image, err := container.Image(ctx)
	if err != nil {
		return nil, err
	}

	if image.Name() != imageInfo.Ref {
		if err := deleteContainer(ctx, container); err != nil {
			return nil, err
		}

		return createContainer(ctx, client, imageInfo)
	}

	return container, nil
}

func createContainer(ctx context.Context, client *containerd.Client,
	imageInfo *ImageInfo) (containerd.Container, error) {

	image, err := ensureImage(ctx, client, imageInfo)
	if err != nil {
		return nil, err
	}

	log.Printf("Creating container %s\n", imageInfo.ID)
	return client.NewContainer(ctx, imageInfo.ID,
		containerd.WithImage(image),
		containerd.WithNewSnapshot(imageInfo.ID+"-snapshot", image),
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
	config *ImageInfo) (containerd.Image, error) {

	image, err := client.GetImage(ctx, config.Ref)
	if err != nil {
		return pullImage(ctx, client, config.Ref)
	}

	return image, nil
}

func pullImage(ctx context.Context, client *containerd.Client, ref string) (containerd.Image, error) {
	log.Printf("Pulling image %s\n", ref)
	return client.Pull(ctx, ref, containerd.WithPullUnpack)
}
