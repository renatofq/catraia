#! /usr/bin/fish

set basepath $PWD
set cmd $argv[1]

function build -d 'build catraia'
    cd $basepath/catraia-net
    go build .
    cd $basepath/catraia-api
    go build .
    cd $basepath
end


function run -d 'run catraia\'s daemons'
    if test (id -u) -ne 0
        echo 'must be root to run catraia'
        return 1
    end

    # load the .env file and exports it's variables
    if test -f .env
        for i in (cat .env)
            set arr (echo $i | tr = \n)
            set -x $arr[1] $arr[2]
        end
    end

    unshare -n $basepath/catraia-net/catraia-net 2>> catraia.log &
    command    $basepath/catraia-api/catraia-api 2>> catraia.log &
end

function stop -d 'stop catraia\'s daemons'
    if test (id -u) -ne 0
        echo 'must be root to stop catraia'
        return 1
    end

    set netps (ps -e | grep catraia-net | head -n 1 | cut -d ' ' -f 1)
    set apips (ps -e | grep catraia-api | head -n 1 | cut -d ' ' -f 1)

    if test -n "$netps"
        kill -SIGTERM $netps[1]
    else
        echo 'catraia-net is not running'
    end

    if test -n "$apips"
        kill -SIGTERM $apips
    else
        echo 'catraia-net is not running'
    end
end

function request  -d 'performs requests to catraia-api' -a req id
    switch $req
        case info
            set method GET
        case deploy
            set method PUT
        case undeploy
            set method DELETE
        case '*'
            echo 'Invalid request.'
            return 1
    end

    if test -z $id
        echo 'Inform the container id'
        return 1
    end

    set url "http://localhost:2077/service/$id"

    curl -X $method $url
end

switch $cmd
    case build
        build
    case run
        run
    case stop
        stop
    case request
        request $argv[2..-1]
    case '*'
        echo 'invalid command:' $cmd
        exit 1
end
