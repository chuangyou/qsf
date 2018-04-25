package runtime

type SuccessJson struct {
	Code   uint32      `json:"code"`
	Result interface{} `json:"result"`
}
