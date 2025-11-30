package file

import (
    "bytes"
    "context"
    "io"
    "os"
    "testing"
)

type fakeConn struct{ files map[string][]byte }

func (f *fakeConn) Exec(ctx context.Context, cmd string, env map[string]string, sudo bool) (string, string, int, error) { return "", "", 0, nil }
func (f *fakeConn) Put(ctx context.Context, src io.Reader, dst string, mode os.FileMode) error {
    b, _ := io.ReadAll(src)
    if f.files == nil { f.files = map[string][]byte{} }
    f.files[dst] = b
    return nil
}
func (f *fakeConn) Get(ctx context.Context, src string) (io.ReadCloser, error) {
    if f.files == nil { return nil, os.ErrNotExist }
    b, ok := f.files[src]
    if !ok { return nil, os.ErrNotExist }
    return io.NopCloser(bytes.NewReader(b)), nil
}

func TestValidateAndCheckApply(t *testing.T) {
    m := mod{}
    args := map[string]any{"path": "/etc/app", "file_name": "app.cfg", "state": "present", "content": "x=1"}
    if err := m.Validate(args); err != nil { t.Fatal(err) }
    fc := &fakeConn{}
    if res, err := m.Check(context.Background(), fc, args); err != nil || !res.Changed { t.Fatalf("expected change, err=%v", err) }
    if _, err := m.Apply(context.Background(), fc, args); err != nil { t.Fatal(err) }
    if res, err := m.Check(context.Background(), fc, args); err != nil || res.Changed { t.Fatalf("expected no change, err=%v", err) }
}

