namespace go user_info

struct GetUserInfoRequest {
    1: string Email (go.tag='json:"username",jsonschema:"description=the email of user"')
} (go.tag='json:"get_user_info"')

struct GetUserInfoResponse {
    1: string Email (go.tag='json:"email",jsonschema:"description=the email of user"')
    2: string Name (go.tag='json:"name,omitempty",jsonschema:"description=the name of user"')
    3: string Avatar (go.tag='json:"avatar,omitempty",jsonschema:"description=the avatar of user"')
    4: string Department (go.tag='json:"department,omitempty",jsonschema:"description=the department of user"')
}

service UserInfoService {
    GetUserInfoResponse GetUserInfo(1: GetUserInfoRequest req) (api.post='/api/v1/user_info/get_user_info')
}