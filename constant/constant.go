package constant

const (
	DEFAULT_ETCD_PATH     = "/qsf.service"
	InitialWindowSize     = 1 << 30
	InitialConnWindowSize = 1 << 30
	MaxSendMsgSize        = 1<<31 - 1
	MaxCallMsgSize        = 1<<31 - 1
)
