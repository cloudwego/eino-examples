/*
 * Copyright 2025 CloudWeGo Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package debug

const PlannerOutput = `初始计划：
1. 使用query_theme_park_opening_hour获取乐园营业时间段，确定可用游玩时长
2. 使用query_park_ticket_price获取三人门票总费用，计算剩余餐饮预算
3. 通过list_locations获取所有区域分布，建立空间认知框架
4. 使用query_performance_info获取所有表演的场次时间、持续时长和所在区域
5. 用query_attraction_info筛选符合身高≤120cm的刺激类游乐设施（过山车/高空项目等）
6. 通过query_attraction_queue_time获取当前各设施预估排队时间
7. 结合query_location_adjacency_info建立区域动线网络，规划最优路径
8. 用query_restaurant_info筛选适合家庭的高性价比餐厅，确保餐饮预算控制
9. 将表演时间作为固定锚点，在表演间隙穿插游乐设施体验
10. 动态调整方案：优先安排低排队时间的优质项目，预留20%缓冲时间应对突发情况
11. 最终整合时需满足：总花费≤2000元、项目间移动时间最短、表演场次不冲突、刺激项目数量最大化`
