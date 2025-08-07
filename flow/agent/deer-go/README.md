
> 本仓库参考 https://github.com/bytedance/deer-flow 完成改写
> 目前完成主要部分的状态图流转
> 
>

# 使用方式
1. 安装python mcp server的依赖，否则运行时会在加载python mcp时卡住。
```bash
cd biz/mcps/python
uv sync
```
2. 进入`conf`文件夹中，复制演示配置文件，并填入配置key
```bash
cp ./conf/deer-go.yaml.1 ./conf/deer-go.yaml
```
3. 运行 `run.sh`，编译并执行。

``` bash
./run.sh
```
4. 如果想配合deerflow的前端运行，需要添加`-s`参数，同时运行deerflow的前端，即可。
``` bash
./run.sh -s
```

5. 如果想采集 trace 和 metrics 埋点信息，可通过配置 `APMPLUS_APP_KEY` 环境变量，来开启埋点采集。采集数据将上报至 APMPlus 。可在 [APMPlus 控制台](https://console.volcengine.com/apmplus-server) 中查看采集到的调用 trace 和 metrics 埋点信息。[火山引擎 APMPlus 文档](https://www.volcengine.com/docs/6431/69092)
``` bash
export APMPLUS_APP_KEY=your_app_key
```

