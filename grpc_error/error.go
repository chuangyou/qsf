package grpc_error

import (
	"strconv"

	"net/http"

	"github.com/chuangyou/qsf/plugin/gateway/runtime"
	dpb "github.com/golang/protobuf/ptypes/duration"
	json "github.com/pquerna/ffjson/ffjson"
	"golang.org/x/net/context"
	epb "google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type ErrorJson struct {
	Code    int32         `json:"code"`
	Message string        `json:"message,omitempty"`
	Details []interface{} `json:"details,omitempty"`
}

//503=服务不可用,500=服务内部错误,401=未登录,403=禁止访问（可带原因）,400=参数错误,429=并发限制,200=OK
func CustomOtherHTTPError(w http.ResponseWriter, _ *http.Request, msg string, code int) {
	w.WriteHeader(code)
	if code == http.StatusNotFound {
		w.Write([]byte("{\"code\":" + strconv.Itoa(int(codes.NotFound)) + ",\"message\":\"" + codes.NotFound.String() + "\"}"))
	} else {
		w.Write([]byte(msg))
	}
}
func CustomHTTPError(ctx context.Context, _ *runtime.ServeMux, marshaler runtime.Marshaler, w http.ResponseWriter, _ *http.Request, err error) {
	var (
		data        []byte
		internalErr error
		httpStauts  int
		errorJson   ErrorJson
	)
	httpStauts, errorJson = ParseError(err)
	if httpStauts == 504 {
		w.WriteHeader(runtime.HTTPStatusFromCode(codes.Unavailable))
		w.Write([]byte("{\"code\":" + strconv.Itoa(int(codes.Unavailable)) + ",\"error\":\"" + codes.Unavailable.String() + "\"}"))
	} else {
		if errorJson.Code == int32(codes.Unavailable) {
			errorJson.Message = codes.Unavailable.String()
		}
		data, internalErr = json.Marshal(errorJson)
		if internalErr != nil {
			w.WriteHeader(runtime.HTTPStatusFromCode(codes.Internal))
			w.Write([]byte("{\"code\":" + strconv.Itoa(int(codes.Internal)) + ",\"error\":\"" + codes.Internal.String() + "\"}"))
		} else {
			w.WriteHeader(httpStauts)
			w.Write(data)
		}
	}
}

//401 UNAUTHENTICATED 由于缺少、无效或过期的令牌，请求未通过身份验证
func Unauthenticated() error {
	var (
		st *status.Status
	)
	st = status.New(codes.Unauthenticated, codes.Unauthenticated.String())
	return st.Err()
}

//403 PERMISSION_DENIED 这可能是因为客户端没有权限
func PermissionDenied(message string) error {
	var (
		st *status.Status
	)
	st = status.New(codes.PermissionDenied, message)
	return st.Err()
}

//400 INVALID_ARGUMENT 客户端使用了错误的参数
func InvalidArgument(field string, description string) error {
	var (
		st       *status.Status
		detSt    *status.Status
		detStErr error
	)
	st = status.New(codes.InvalidArgument, codes.InvalidArgument.String())
	detSt, detStErr = st.WithDetails(&epb.BadRequest{
		FieldViolations: []*epb.BadRequest_FieldViolation{
			{
				Field:       field,
				Description: description,
			},
		},
	})
	if detStErr == nil {
		return detSt.Err()
	} else {
		return st.Err()
	}
}

//429 RESOURCE_EXHAUSTED  超过资源限额或频率限制
func ResourceExhausted(subject, description string) error {
	var (
		st       *status.Status
		detSt    *status.Status
		detStErr error
	)
	st = status.New(codes.ResourceExhausted, codes.ResourceExhausted.String())
	detSt, detStErr = st.WithDetails(&epb.QuotaFailure{
		Violations: []*epb.QuotaFailure_Violation{
			{
				Subject:     subject,
				Description: description,
			},
		},
	},
		&epb.RetryInfo{
			RetryDelay: &dpb.Duration{Seconds: 30},
		},
	)
	if detStErr == nil {
		return detSt.Err()
	} else {
		return st.Err()
	}
}

//500 INTERNAL 服务端内部错误，一般是 BUG
func Internal() error {
	var (
		st *status.Status
	)
	st = status.New(codes.Internal, codes.Internal.String())
	return st.Err()
}

//503 UNAVAILABLE  服务端不可用
func Unavailable() error {
	var (
		st *status.Status
	)
	st = status.New(codes.Unavailable, codes.Unavailable.String())
	return st.Err()
}

func ParseError(err error) (httpStatus int, data ErrorJson) {
	var (
		s = new(status.Status)
	)
	s = status.Convert(err)
	httpStatus = runtime.HTTPStatusFromCode(s.Code())
	data.Code = int32(s.Code())
	data.Message = s.Message()
	for _, d := range s.Details() {
		switch info := d.(type) {
		case *epb.RetryInfo:
			data.Details = append(data.Details, info)
		case *epb.DebugInfo:
			data.Details = append(data.Details, info)
		case *epb.QuotaFailure:
			data.Details = append(data.Details, info)
		case *epb.BadRequest:
			data.Details = append(data.Details, info)
		case *epb.RequestInfo:
			data.Details = append(data.Details, info)
		case *epb.ResourceInfo:
			data.Details = append(data.Details, info)
		case *epb.Help:
			data.Details = append(data.Details, info)
		case *epb.LocalizedMessage:
			data.Details = append(data.Details, info)
		default:
			continue
		}
	}
	return
}
