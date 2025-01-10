module github.com/cloudwego/eino-examples

go 1.22.0

toolchain go1.22.1

require (
	code.byted.org/gopkg/logs/v2 v2.2.0-beta.9
	code.byted.org/kite/kitex v1.17.2
	github.com/apache/thrift v0.19.0
	github.com/bytedance/sonic v1.12.3
	github.com/cloudwego/eino v0.3.4
	github.com/cloudwego/eino-ext/components/model/ollama v0.0.0-20250103071650-0c38eea04df7
	github.com/cloudwego/eino-ext/components/model/openai v0.0.0-20250103071650-0c38eea04df7
	github.com/cloudwego/eino-ext/components/retriever/volc_vikingdb v0.0.0-20250106073650-ed838398894a
	github.com/cloudwego/eino-ext/components/tool/googlesearch v0.0.0-20250103071650-0c38eea04df7
	github.com/cloudwego/eino-ext/devops v0.0.0-20250109073820-fb2d91009583
	github.com/cloudwego/kitex v0.11.3
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc
	github.com/getkin/kin-openapi v0.118.0
	github.com/ollama/ollama v0.3.0
)

require (
	cloud.google.com/go/auth v0.10.0 // indirect
	cloud.google.com/go/auth/oauth2adapt v0.2.5 // indirect
	cloud.google.com/go/compute/metadata v0.5.2 // indirect
	code.byted.org/aiops/apm_vendor_byted v0.0.27 // indirect
	code.byted.org/aiops/metrics_codec v0.0.24 // indirect
	code.byted.org/aiops/monitoring-common-go v0.0.4 // indirect
	code.byted.org/bcc/bcc-go-client v0.1.37 // indirect
	code.byted.org/bcc/bcc-go-client/internal/sidecar/idl v0.0.4 // indirect
	code.byted.org/bcc/conf_engine v0.0.0-20230510030051-32fb55f74cf1 // indirect
	code.byted.org/bcc/pull_json_model v1.0.22 // indirect
	code.byted.org/bcc/tools v0.0.21 // indirect
	code.byted.org/bytedtrace/bytedtrace-client-go v1.2.2 // indirect
	code.byted.org/bytedtrace/bytedtrace-common/go v0.0.13 // indirect
	code.byted.org/bytedtrace/bytedtrace-conf-provider-client-go v0.0.27 // indirect
	code.byted.org/bytedtrace/bytedtrace-gls-switch v1.2.0 // indirect
	code.byted.org/bytedtrace/interface-go v1.0.20 // indirect
	code.byted.org/bytedtrace/serializer-go v1.0.1-pre // indirect
	code.byted.org/duanyi.aster/gopkg v0.0.4 // indirect
	code.byted.org/gopkg/apm_vendor_interface v0.0.3 // indirect
	code.byted.org/gopkg/asynccache v0.0.0-20210422090342-26f94f7676b8 // indirect
	code.byted.org/gopkg/consul v1.2.6 // indirect
	code.byted.org/gopkg/ctxvalues v0.7.0 // indirect
	code.byted.org/gopkg/debug v0.10.1 // indirect
	code.byted.org/gopkg/env v1.6.14 // indirect
	code.byted.org/gopkg/etcd_util v2.3.3+incompatible // indirect
	code.byted.org/gopkg/etcdproxy v0.1.1 // indirect
	code.byted.org/gopkg/logid v0.0.0-20241008043456-230d03adb830 // indirect
	code.byted.org/gopkg/logs v1.2.23 // indirect
	code.byted.org/gopkg/metainfo v0.1.1 // indirect
	code.byted.org/gopkg/metrics v1.4.25 // indirect
	code.byted.org/gopkg/metrics/v3 v3.1.31 // indirect
	code.byted.org/gopkg/metrics/v4 v4.1.4 // indirect
	code.byted.org/gopkg/metrics_core v0.0.39 // indirect
	code.byted.org/gopkg/mockito v1.3.0 // indirect
	code.byted.org/gopkg/net2 v1.5.0 // indirect
	code.byted.org/gopkg/stats v1.2.12 // indirect
	code.byted.org/gopkg/tccclient v1.5.0-beta.10 // indirect
	code.byted.org/gopkg/thrift v1.14.1 // indirect
	code.byted.org/kite/rpal v0.1.19 // indirect
	code.byted.org/lang/trace v0.0.3 // indirect
	code.byted.org/lidar/profiler v0.4.4 // indirect
	code.byted.org/lidar/profiler/kitex v0.0.0-20240515095433-9c7e047c4f64 // indirect
	code.byted.org/log_market/gosdk v0.0.0-20230524072203-e069d8367314 // indirect
	code.byted.org/log_market/loghelper v0.1.11 // indirect
	code.byted.org/log_market/tracelog v0.1.5 // indirect
	code.byted.org/log_market/ttlogagent_gosdk v0.0.6 // indirect
	code.byted.org/log_market/ttlogagent_gosdk/v4 v4.0.53 // indirect
	code.byted.org/middleware/fic_client v0.2.8 // indirect
	code.byted.org/security/go-spiffe-v2 v1.0.8 // indirect
	code.byted.org/security/memfd v0.0.2 // indirect
	code.byted.org/security/sensitive_finder_engine v0.3.18 // indirect
	code.byted.org/security/zti-jwt-helper-golang v1.0.17 // indirect
	code.byted.org/service_mesh/shmipc v0.2.16 // indirect
	code.byted.org/trace/trace-client-go v1.3.7 // indirect
	code.byted.org/ttarch/byteconf-cel-go v0.0.3 // indirect
	github.com/Knetic/govaluate v3.0.1-0.20171022003610-9aa49832a739+incompatible // indirect
	github.com/agiledragon/gomonkey/v2 v2.12.0 // indirect
	github.com/antonmedv/expr v1.15.5 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/bits-and-blooms/bitset v1.13.0 // indirect
	github.com/bits-and-blooms/bloom/v3 v3.6.0 // indirect
	github.com/blang/semver/v4 v4.0.0 // indirect
	github.com/bytedance/gopkg v0.1.1 // indirect
	github.com/bytedance/sonic/loader v0.2.0 // indirect
	github.com/caarlos0/env/v6 v6.10.1 // indirect
	github.com/cenkalti/backoff/v4 v4.2.1 // indirect
	github.com/cloudwego/base64x v0.1.4 // indirect
	github.com/cloudwego/configmanager v0.2.2 // indirect
	github.com/cloudwego/dynamicgo v0.4.0 // indirect
	github.com/cloudwego/eino-ext/libs/acl/openai v0.0.0-20241224084133-4395591bc455 // indirect
	github.com/cloudwego/fastpb v0.0.5 // indirect
	github.com/cloudwego/frugal v0.2.0 // indirect
	github.com/cloudwego/gopkg v0.1.2 // indirect
	github.com/cloudwego/iasm v0.2.0 // indirect
	github.com/cloudwego/localsession v0.0.2 // indirect
	github.com/cloudwego/netpoll v0.6.4 // indirect
	github.com/cloudwego/runtimex v0.1.0 // indirect
	github.com/cloudwego/thriftgo v0.3.17 // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/fatih/structtag v1.2.0 // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/go-jose/go-jose/v3 v3.0.3 // indirect
	github.com/go-kit/log v0.2.1 // indirect
	github.com/go-logfmt/logfmt v0.6.0 // indirect
	github.com/go-logr/logr v1.4.2 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-ole/go-ole v1.3.0 // indirect
	github.com/go-openapi/jsonpointer v0.19.5 // indirect
	github.com/go-openapi/swag v0.19.5 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/google/pprof v0.0.0-20240727154555-813a5fbdbec8 // indirect
	github.com/google/s2a-go v0.1.8 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/googleapis/enterprise-certificate-proxy v0.3.4 // indirect
	github.com/googleapis/gax-go/v2 v2.13.0 // indirect
	github.com/goph/emperror v0.17.2 // indirect
	github.com/gorilla/mux v1.8.1 // indirect
	github.com/hashicorp/golang-lru v1.0.2 // indirect
	github.com/hbollon/go-edlib v1.6.0 // indirect
	github.com/iancoleman/strcase v0.3.0 // indirect
	github.com/invopop/yaml v0.3.1 // indirect
	github.com/jhump/protoreflect v1.8.2 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/klauspost/compress v1.17.9 // indirect
	github.com/klauspost/cpuid/v2 v2.2.7 // indirect
	github.com/kr/pretty v0.3.1 // indirect
	github.com/magiconair/properties v1.8.7 // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/mattn/go-colorable v0.1.12 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/gls v0.0.0-20220109145502-612d0167dce5 // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/mohae/deepcopy v0.0.0-20170929034955-c48cc78d4826 // indirect
	github.com/nikolalohinski/gonja v1.5.3 // indirect
	github.com/opentracing/opentracing-go v1.2.1-0.20210726034734-bdbb7cc3a1c0 // indirect
	github.com/pelletier/go-toml/v2 v2.2.2 // indirect
	github.com/perimeterx/marshmallow v1.1.5 // indirect
	github.com/pierrec/lz4/v4 v4.1.15 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/power-devops/perfstat v0.0.0-20240221224432-82ca36839d55 // indirect
	github.com/rogpeppe/go-internal v1.12.0 // indirect
	github.com/sashabaranov/go-openai v1.32.5 // indirect
	github.com/shirou/gopsutil/v3 v3.24.2 // indirect
	github.com/sirupsen/logrus v1.9.3 // indirect
	github.com/slongfield/pyfmt v0.0.0-20220222012616-ea85ff4c361f // indirect
	github.com/stretchr/testify v1.9.0 // indirect
	github.com/tidwall/gjson v1.17.3 // indirect
	github.com/tidwall/match v1.1.1 // indirect
	github.com/tidwall/pretty v1.2.1 // indirect
	github.com/twitchyliquid64/golang-asm v0.15.1 // indirect
	github.com/vmihailenco/msgpack/v5 v5.4.1 // indirect
	github.com/vmihailenco/tagparser/v2 v2.0.0 // indirect
	github.com/volcengine/volc-sdk-golang v1.0.182 // indirect
	github.com/yargevad/filepathx v1.0.0 // indirect
	github.com/yusufpapurcu/wmi v1.2.4 // indirect
	github.com/zeebo/errs v1.3.0 // indirect
	go.opencensus.io v0.24.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.54.0 // indirect
	go.opentelemetry.io/otel v1.29.0 // indirect
	go.opentelemetry.io/otel/metric v1.29.0 // indirect
	go.opentelemetry.io/otel/trace v1.29.0 // indirect
	go4.org/unsafe/assume-no-moving-gc v0.0.0-20231121144256-b99613f794b6 // indirect
	golang.org/x/arch v0.11.0 // indirect
	golang.org/x/crypto v0.31.0 // indirect
	golang.org/x/exp v0.0.0-20240222234643-814bf88cf225 // indirect
	golang.org/x/net v0.33.0 // indirect
	golang.org/x/oauth2 v0.23.0 // indirect
	golang.org/x/sync v0.10.0 // indirect
	golang.org/x/sys v0.28.0 // indirect
	golang.org/x/text v0.21.0 // indirect
	golang.org/x/time v0.7.0 // indirect
	google.golang.org/api v0.204.0 // indirect
	google.golang.org/genproto v0.0.0-20241021214115-324edc3d5d38 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20241015192408-796eee8c2d53 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20241021214115-324edc3d5d38 // indirect
	google.golang.org/grpc v1.67.1 // indirect
	google.golang.org/protobuf v1.35.1 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/apache/thrift => github.com/apache/thrift v0.13.0
