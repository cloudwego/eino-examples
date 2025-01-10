/*
 * Copyright 2024 CloudWeGo Authors
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

package main

import (
	"context"

	"github.com/bytedance/sonic"

	"github.com/cloudwego/eino/components/tool/utils"
	"code.byted.org/gopkg/logs/v2"

	"github.com/cloudwego/eino-examples/components/tool/rpc_tool/kitex_gen/user_info"
	"github.com/cloudwego/eino-examples/components/tool/rpc_tool/kitex_gen/user_info/userinfoservice"
)

var (
	userInfoCli userinfoservice.Client
)

func init() {
	var err error
	var psm = "eino.user.info"
	userInfoCli, err = userinfoservice.NewClient(psm)
	if err != nil {
		panic(err)
	}
}

func main() {

	ctx := context.Background()

	tl, err := utils.InferTool("get_user_info", "get user info", getUserInfo)
	if err != nil {
		logs.Errorf("InferTool failed: %v", err)
		return
	}

	info, err := tl.Info(ctx)
	if err != nil {
		logs.Errorf("Get ToolInfo failed: %v", err)
		return
	}
	infoJSON, err := sonic.MarshalIndent(info, "", "  ")
	if err != nil {
		logs.Errorf("MarshalIndent failed: %v", err)
		return
	}
	logs.Infof("ToolInfo: \n%v", string(infoJSON))

	content, err := tl.InvokableRun(ctx, `{"email":"bruce_lee@bytedance.com"}`)
	if err != nil {
		logs.Errorf("InvokableRun failed: %v", err)
		return
	}
	logs.Infof("Content: %v", content)
}

func getUserInfo(ctx context.Context, req *user_info.GetUserInfoRequest) (resp *user_info.GetUserInfoResponse, err error) {
	// return userInfoCli.GetUserInfo(ctx, req)

	// mock response
	return &user_info.GetUserInfoResponse{
		Email: req.GetEmail(),
		Name:  "Bruce Lee",
	}, nil
}
