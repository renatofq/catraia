#! /usr/bin/fish

function catraia -d 'curl based catraia client' -a request id
    switch $request
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
        return
    end

    set url "http://localhost:2077/service/$id"

    curl -X $method $url
end
