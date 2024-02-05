#!/bin/bash

echo "Creating symlink to $1 at $2"

ln -s -f -d "$1" "$2"

echo "ln -s $1 $2" >> /tmp/symlink.log
echo "-------------------" >> /tmp/symlink.log

# Dump env to /tmp/env.log
env | grep radarr >> /tmp/env_radarr.log
echo "-------------------" >> /tmp/env_radarr.log

env | grep sonarr >> /tmp/env_sonarr.log
echo "-------------------" >> /tmp/env_sonarr.log
