#!/usr/bin/env bash
################################################################################
# This script is to facilitate starting and stopping DMoni manager daemon.
#
# Usage:
#   $ ./dmoni_manager.sh (start|stop)
#
################################################################################

#some variables
usage="Usage: dmoni_manager.sh (start|stop)"
log=/tmp/dmoni_manager.out
pid=/tmp/dmoni_manager.pid
STOP_TIMEOUT=2

if [ "$#" -ne "1" ]; then
    echo $usage
    exit 1
fi

if [ -z "$DMONIPATH" ]; then
    echo DMONIPATH is not set
    exit 1
fi

if [ -z "$DMONI_STORAGE" ]; then
    echo DMONI_STORAGE is not set
    exit 1
fi

if [ -z "$DMONI_HOST" ]; then
    DMONI_HOST=$(hostname)
fi

if [ -z "$DMONI_APP_PORT" ]; then
    DMONI_APP_PORT=5500
fi

if [ -z "$DMONI_NODE_PORT" ]; then
    DMONI_NODE_PORT=5300
fi

case "$1" in
    start)

        # Check if manager is already running
        if [ -f $pid ]; then
            if kill -0 $(cat $pid) > /dev/null 2>&1; then
                echo "DMoni manager running as process $(cat $pid). Stop it first."
                exit 1
            fi
        fi

        echo starting DMoni manager
        nohup dmoni manager --storage $DMONI_STORAGE \
            --host $DMONI_HOST \
            --app-port $DMONI_APP_PORT \
            --node-port $DMONI_NODE_PORT \
            >> $log 2>&1 < /dev/null &
        echo $! > $pid
        sleep 1
        head $log
        ;;

    stop)

        if [ -f $pid ]; then
            echo stopping DMoni manager
            TARGET_PID=$(cat $pid)
            if kill -0 $TARGET_PID > /dev/null 2>&1; then
                kill $TARGET_PID
                sleep $STOP_TIMEOUT
                if kill -0 $TARGET_PID > /dev/null 2>&1; then
                    echo "did not stop gracefully after $STOP_TIMEOUT seconds: killing with kill -9"
                    kill -9 $TARGET_PID
                fi
            else
                echo no DMoni manager to stop
            fi
            rm -f $pid
        else
            echo no DMoni manager to stop
        fi
        ;;

    *)
        echo $usage
        exit 1
        ;;
esac
