#!/usr/bin/env bash
################################################################################
# This script is to facilitate starting and stopping DMoni agent daemon.
#
# Usage:
#   $ ./dmoni_agent.sh (start|stop)
#
################################################################################

# variables must be specified by user
DMONI_STORAGE=
DMONI_MANAGER=

# variables that can be customized. 
# if a value is not given, the default value is used.
DMONI_HOST=
DMONI_PORT=
DMONI_MANAGER_PORT=

# some variables
usage="Usage: dmoni_agent.sh (start|stop)"
log=/tmp/dmoni_agent.out
pid=/tmp/dmoni_agent.pid
STOP_TIMEOUT=2

if [ "$#" -ne "1" ]; then
    echo $usage
    exit 1
fi

if [ -z "$DMONIPATH" ]; then
    echo DMONIPATH is not set
    exit 1
fi

if [ -z "$DMONI_HOST" ]; then
    DMONI_HOST=$(hostname)
fi

if [ -z "$DMONI_PORT" ]; then
    DMONI_PORT=5301
fi

if [ -z "$DMONI_MANAGER_PORT" ]; then
    DMONI_MANAGER_PORT=5300
fi

case "$1" in
    start)

        # Check if agent is already running
        if [ -f $pid ]; then
            if kill -0 $(cat $pid) > /dev/null 2>&1; then
                echo "DMoni agent running as process $(cat $pid). Stop it first."
                exit 1
            fi
        fi

        echo starting DMoni agent
        nohup dmoni agent --storage $DMONI_STORAGE \
            --manager $DMONI_MANAGER --manager-port $DMONI_MANAGER_PORT \
            --host $DMONI_HOST --port $DMONI_PORT \
            >> $log 2>&1 < /dev/null &
        echo $! > $pid
        sleep 1
        head $log
        ;;

    stop)

        if [ -f $pid ]; then
            echo stopping DMoni agent
            TARGET_PID=$(cat $pid)
            if kill -0 $TARGET_PID > /dev/null 2>&1; then
                kill $TARGET_PID
                sleep $STOP_TIMEOUT
                if kill -0 $TARGET_PID > /dev/null 2>&1; then
                    echo "did not stop gracefully after $STOP_TIMEOUT seconds: killing with kill -9"
                    kill -9 $TARGET_PID
                fi
            else
                echo no DMoni agent to stop
            fi
            rm -f $pid
        else
            echo no DMoni agent to stop
        fi
        ;;

    *)
        echo $usage
        exit 1
        ;;
esac
