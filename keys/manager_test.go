package keys

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/keks/testops"
	"github.com/stretchr/testify/require"
	"modernc.org/kv"

	"go.cryptoscope.co/librarian"
)

type opDo func(t *testing.T, env interface{})

func (op opDo) Do(t *testing.T, env interface{}) {
	op(t, env)
}

func TestManager(t *testing.T) {
	tDir, err := ioutil.TempDir(".", "test-manager-")
	require.NoError(t, err, "mk temp dir")

	defer func() {
		require.NoError(t, os.RemoveAll(tDir), "rm temp dir")
	}()

	var (
		idx librarian.SeqSetterIndex
		db  *kv.DB
		mgr Manager
		ctx = context.Background()
	)

	tcs := []testops.TestCase{
		testops.TestCase{
			Name: "compound test", // TODO: split this into smaller tests
			Ops: []testops.Op{
				opDBCreate{
					Name: filepath.Join(tDir, "testdb"),
					Opts: &kv.Options{},
					DB:   &db,
				},
				opIndexNew{
					DB:    &db,
					Type:  Keys(nil),
					Index: &idx,
				},
				opDo(func(t *testing.T, env interface{}) {
					mgr = Manager{idx}
				}),
				opManagerAddKey{
					Mgr:  &mgr,
					ID:   ID("test"),
					Type: "IDSign",
					Key:  Key("topsecret"),
					Ctx:  &ctx,
				},
				opIndexGet{
					Index: &idx,
					Addr: librarian.Addr([]byte{
						6, 0, // type is 6 byte long
						'I', 'D', 'S', 'i', 'g', 'n', // "IDSign"
						4, 0, // db key is four byte long
						't', 'e', 's', 't', // "test"
					}),

					ExpValue: Keys{Key("topsecret")},
				},
				opDBGet{
					DB: &db,
					Key: []byte{
						6, 0, // type is 6 byte long
						'I', 'D', 'S', 'i', 'g', 'n', // "IDSign"
						4, 0, // db key is four byte long
						't', 'e', 's', 't', // "test"
					},

					ExpValue: func(ks Keys) []byte {
						var strs = make([]string, len(ks))
						for i := range ks {
							strs[i] = base64.StdEncoding.EncodeToString((ks)[i])
						}

						exp, err := json.Marshal(strs)
						require.NoError(t, err, "json encode of test string")
						return exp
					}(Keys{Key("topsecret")}),
				},
				opManagerAddKey{
					Mgr:  &mgr,
					ID:   ID("test"),
					Type: "IDSign",
					Key:  Key("alsosecret"),
					Ctx:  &ctx,
				},
				opIndexGet{
					Index: &idx,
					Addr: librarian.Addr([]byte{
						6, 0, // type is 6 byte long
						'I', 'D', 'S', 'i', 'g', 'n', // "IDSign"
						4, 0, // db key is four byte long
						't', 'e', 's', 't', // "test"
					}),

					ExpValue: Keys{Key("topsecret"), Key("alsosecret")},
				},
				opDBGet{
					DB: &db,
					Key: []byte{
						6, 0, // type is 6 byte long
						'I', 'D', 'S', 'i', 'g', 'n', // "IDSign"
						4, 0, // db key is four byte long
						't', 'e', 's', 't', // "test"
					},

					ExpValue: func(ks Keys) []byte {
						var strs = make([]string, len(ks))
						for i := range ks {
							strs[i] = base64.StdEncoding.EncodeToString((ks)[i])
						}

						exp, err := json.Marshal(strs)
						require.NoError(t, err, "json encode of test string")
						return exp
					}(Keys{Key("topsecret"), Key("alsosecret")}),
				},
				opManagerSetKey{
					Mgr: &mgr,
					ID:  ID("foo"),
					Key: Key("bar"),
					Ctx: &ctx,
				},
				opDBGet{
					DB: &db,
					Key: []byte{
						0, 0, // empty type
						3, 0, // db key is four byte long
						'f', 'o', 'o', // "foo"
					},

					ExpValue: func(ks Keys) []byte {
						var strs = make([]string, len(ks))
						for i := range ks {
							strs[i] = base64.StdEncoding.EncodeToString((ks)[i])
						}

						exp, err := json.Marshal(strs)
						require.NoError(t, err, "json encode of test string")
						return exp
					}(Keys{Key("bar")}),
				},
				opManagerGetKeys{
					Mgr:     &mgr,
					ID:      ID("test"),
					Type:    "IDSign",
					Ctx:     &ctx,
					ExpKeys: Keys{Key("topsecret"), Key("alsosecret")},
				},
				opManagerGetKeys{
					Mgr:     &mgr,
					ID:      ID("foo"),
					Ctx:     &ctx,
					ExpKeys: Keys{Key("bar")},
				},
				opManagerRmKeys{
					Mgr:  &mgr,
					ID:   ID("test"),
					Type: "IDSign",
					Ctx:  &ctx,
				},
				opManagerGetKeys{
					Mgr:    &mgr,
					ID:     ID("test"),
					Type:   "IDSign",
					Ctx:    &ctx,
					ExpErr: fmt.Sprintf("no such key at (IDSign, %x)", "test"),
				},
			},
		},
		{
			Name: "idxKey encode - short buffer",
			Ops: []testops.Op{
				opDBKeyEncode{
					Key: &idxKey{
						t:  "type",
						id: ID("test"),
					},
					BufLen: 1,
					ExpErr: "buffer too short: need 12, got 1",
				},
			},
		},
		{
			Name: "idxKey decode",
			Ops: []testops.Op{
				opDBKeyDecode{
					Bytes:  []byte{0}, // only one byte, too short to read type length
					ExpErr: "data too short to read type length",
				},
				opDBKeyDecode{
					Bytes:  []byte{4, 0, 1}, // buffer too short to read type of length 4
					ExpErr: "invalid key - claimed type length exceeds buffer",
				},
				opDBKeyDecode{
					Bytes:  []byte{4, 0, 't', 'e', 's', 't', 6}, // buffer too short to read type of length 4 plus 2 byte size
					ExpErr: "invalid key - claimed type length exceeds buffer",
				},
				opDBKeyDecode{
					Bytes: []byte{
						4, 0, 't', 'e', 's', 't',
						6, 0, 'f', 'o', 'o', 'b'}, // buffer too short to read id of length 6
					ExpErr: "invalid key - claimed id length exceeds buffer",
				},
				opDBKeyDecode{
					Bytes: []byte{
						4, 0, 't', 'e', 's', 't',
						6, 0, 'f', 'o', 'o', 'b', 'a', 'r',
					},
					ExpKey: &idxKey{t: "test", id: ID("foobar")},
				},
			},
		},
		{
			Name: "encode idxKey",
			Ops: []testops.Op{
				opDBKeyEncode{
					Key: &idxKey{
						t:  "type",
						id: ID("test"),
					},
					ExpData: []byte{
						4, 0,
						't', 'y', 'p', 'e',
						4, 0,
						't', 'e', 's', 't',
					},
				},
				opDBKeyEncode{
					Key: &idxKey{
						t:  "group",
						id: ID("test"),
					},
					ExpData: []byte{
						5, 0,
						'g', 'r', 'o', 'u', 'p',
						4, 0,
						't', 'e', 's', 't',
					},
				},
			},
		},
	}

	testops.Run(t, []testops.Env{testops.Env{
		Name: "Keys",
		Func: func(tc testops.TestCase) (func(*testing.T), error) {
			return tc.Runner(nil), nil
		},
	}}, tcs)
}