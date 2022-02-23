#!/usr/bin/env bash

cleanup() {
    echo "exiting"
}

sighup() {
    echo "SIGHUP received"
    cleanup
}

sigint() {
    echo "SIGINT received"
    cleanup
}

sigterm() {
    echo "SIGTERM received"
    cleanup
}

run() {
    trap "sighup; exit" SIGHUP
    trap "sigint; exit" SIGINT
    trap "sigterm; exit" SIGTERM

    echo "PID: $$"

    while true; do
        echo "waiting for a signal..."
        sleep 1;
    done
}

run
