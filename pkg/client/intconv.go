package client

import "math"

// int32PtrFromInt64 returns id as *int32 when it fits a signed 32-bit integer.
// Values outside that range yield nil so OpenAPI filter fields are left unset.
func int32PtrFromInt64(id int64) *int32 {
	if id > math.MaxInt32 || id < math.MinInt32 {
		return nil
	}

	v := int32(id)

	return &v
}
