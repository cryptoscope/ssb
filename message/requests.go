// SPDX-License-Identifier: MIT

package message

import (
	"fmt"
	"strings"

	refs "go.mindeco.de/ssb-refs"
)

type WhoamiReply struct {
	ID *refs.FeedRef `json:"id"`
}

func NewCreateHistArgsFromMap(argMap map[string]interface{}) (*CreateHistArgs, error) {

	// could reflect over qrys fiields but meh - compiler knows better
	var qry CreateHistArgs
	for k, v := range argMap {
		switch k = strings.ToLower(k); k {
		case "live", "keys", "values", "reverse", "asjson", "private":
			b, ok := v.(bool)
			if !ok {
				return nil, fmt.Errorf("ssb/message: not a bool for %s", k)
			}
			switch k {
			case "live":
				qry.Live = b
			case "keys":
				qry.Keys = b
			case "values":
				qry.Values = b
			case "reverse":
				qry.Reverse = b
			case "asjson":
				qry.AsJSON = b
			case "private":
				qry.Private = b
			}

		case "type", "id":
			val, ok := v.(string)
			if !ok {
				return nil, fmt.Errorf("ssb/message: not string (but %T) for %s", v, k)
			}
			switch k {
			case "id":
				var err error
				qry.ID, err = refs.ParseFeedRef(val)
				if err != nil {
					return nil, fmt.Errorf("ssb/message: not a feed ref: %w", err)
				}
			}
		case "seq", "limit", "gt", "lt":
			n, ok := v.(float64)
			if !ok {
				return nil, fmt.Errorf("ssb/message: not a float64(%T) for %s", v, k)
			}
			switch k {
			case "seq":
				qry.Seq = int64(n)
			case "limit":
				qry.Limit = int64(n)
			case "gt":
				qry.Gt = int64(n)
			case "lt":
				qry.Lt = int64(n)
			}
		}
	}

	if qry.Limit == 0 {
		qry.Limit = -1
	}

	return &qry, nil
}

type CommonArgs struct {
	Keys   bool `json:"keys"` // can't omit this falsy value, the JS-stack stack assumes true if it's not there
	Values bool `json:"values,omitempty"`
	Live   bool `json:"live,omitempty"`

	Private bool `json:"private,omitempty"`
}

type StreamArgs struct {
	Limit int64 `json:"limit,omitempty"`

	Gt int64 `json:"gt,omitempty"`
	Lt int64 `json:"lt,omitempty"`

	Reverse bool `json:"reverse,omitempty"`
}

// CreateHistArgs defines the query parameters for the createHistoryStream rpc call
type CreateHistArgs struct {
	CommonArgs
	StreamArgs

	ID  *refs.FeedRef `json:"id,omitempty"`
	Seq int64         `json:"seq,omitempty"`

	AsJSON bool `json:"asJSON,omitempty"`
}

// CreateLogArgs defines the query parameters for the createLogStream rpc call
type CreateLogArgs struct {
	CommonArgs
	StreamArgs

	Seq int64 `json:"seq"`
}

// MessagesByTypeArgs defines the query parameters for the messagesByType rpc call
type MessagesByTypeArgs struct {
	CommonArgs
	StreamArgs
	Type string `json:"type"`
}

type TanglesArgs struct {
	CommonArgs
	StreamArgs

	Root *refs.MessageRef `json:"root"`

	// indicate the v2 subtangle (group, ...)
	// empty string for v1 tangle
	Name string `json:"name"`
}
