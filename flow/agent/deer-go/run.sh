#!/usr/bin/env bash
RUN_NAME="deer-flow-go"

mkdir -p output/bin output/conf
cp conf/* output/conf/
go build -o output/bin/${RUN_NAME}
chmod +x output/bin/${RUN_NAME}
# 将所有参数传递给程序
./output/bin/${RUN_NAME} "$@"