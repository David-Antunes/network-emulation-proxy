#!/bin/sh

if [ -z "$NAMESPACE" ];
  then
    echo "NAMESPACE variable not set"
    exit 1
fi
  
nsenter --net=$NAMESPACE "./proxy"
