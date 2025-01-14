
> Eino 发音：美 / 'aino /，近似音: i know，有希望应用程序达到 "i know" 的愿景

# Eino  是什么

> 💡
> Go AI 集成组件的研发框架。

Eino 旨在提供 Golang 语言的 AI 应用开发框架。 Eino 参考了开源社区中诸多优秀的 AI 应用开发框架，例如 LangChain、LangGraph、LlamaIndex 等，提供了更符合 Golang 编程习惯的 AI 应用开发框架。

Eino 提供了丰富的辅助 AI 应用开发的**原子组件**、**集成组件**、**组件编排**、**切面扩展**等能力，可以帮助开发者更加简单便捷地开发出架构清晰、易维护、高可用的 AI 应用。

以 React Agent 为例：

- 提供了 ChatModel、ToolNode、PromptTemplate 等常用组件，并且业务可定制、可扩展。
- 可基于现有组件进行灵活编排，产生集成组件，在业务服务中使用。

![](/img/eino/react_agent_graph.png)

# Eino 组件

> Eino 的其中一个目标是：搜集和完善 AI 应用场景下的组件体系，让业务可轻松找到一些通用的 AI 组件，方便业务的迭代。

Eino 会围绕 AI 应用的场景，提供具有比较好的抽象的组件，并围绕该抽象提供一些常用的实现。

- Eino 组件的抽象定义在：[eino/components](https://github.com/cloudwego/eino/tree/main/components)
- Eino 组件的实现在：[eino-ext/components](https://github.com/cloudwego/eino-ext/tree/main/components)

# Eino 应用场景

得益于 Eino 轻量化和内场亲和属性，用户只需短短数行代码就能给你的存量微服务引入强力的大模型能力，让传统微服务进化出 AI 基因。

可能大家听到【Graph 编排】这个词时，第一反应就是将整个应用接口的实现逻辑进行分段、分层的逻辑拆分，并将其转换成可编排的 Node。 这个过程中遇到的最大问题就是**长距离的上下文传递(跨 Node 节点的变量传递)**问题，无论是使用 Graph/Chain 的 State，还是使用 Options 透传，整个编排过程都极其复杂，远没有直接进行函数调用简单。

基于当前的 Graph 编排能力，适合编排的场景具有如下几个特点：

- 整体是围绕模型的语义处理相关能力。这里的语义不限模态
- 编排产物中有极少数节点是 Session 相关的。整体来看，绝大部分节点没有类似用户/设备等不可枚举地业务实体粒度的处理逻辑

  - 无论是通过 Graph/Chain 的 State、还是通过 CallOptions，对于读写或透传用户/设备粒度的信息的方式，均不简便
- 需要公共的切面能力，基于此建设观测、限流、评测等横向治理能力

编排的意义是什么： 把长距离的编排元素上下文以固定的范式进行聚合控制和呈现。

**整体来说，“Graph 编排”适用的场景是： 业务定制的 AI 集成组件。  ****即把 AI 相关的原子能力，进行灵活编排****，提供简单易用的场景化的 AI 组件。 并且该 AI 组件中，具有统一且完整的横向治理能力。**

- 推荐的使用方式

![](/img/eino/recommend_way_of_handler.png)

- 挑战较大的方式 -- 【业务全流程的节点编排】
  - Biz Handler 一般重业务逻辑，轻数据流，比较适合函数栈调用的方式进行开发
    - 如果采用图编排的方式进行逻辑划分与组合，会增大业务逻辑开发的难度

![](/img/eino/big_challenge_graph.png)
