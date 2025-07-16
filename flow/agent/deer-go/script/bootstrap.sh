#!/bin/bash
CURDIR=$(cd $(dirname $0); pwd)
BinaryName=deer-flow-go
echo "$CURDIR/bin/${BinaryName}"
exec $CURDIR/bin/${BinaryName}