package conn

import (
    "context"
    "fmt"
    "io"
    "net"
    "os"
    "time"

    "github.com/pkg/sftp"
    "golang.org/x/crypto/ssh"
)

type Conn interface {
    Exec(ctx context.Context, cmd string, env map[string]string, sudo bool) (string, string, int, error)
    Put(ctx context.Context, src io.Reader, dst string, mode os.FileMode) error
    Get(ctx context.Context, src string) (io.ReadCloser, error)
}

type SSHConn struct {
    client *ssh.Client
    sftp   *sftp.Client
}

func Dial(user, addr string, key []byte, timeout time.Duration) (*SSHConn, error) {
    signer, err := ssh.ParsePrivateKey(key)
    if err != nil {
        return nil, err
    }
    cfg := &ssh.ClientConfig{
        User:            user,
        Auth:            []ssh.AuthMethod{ssh.PublicKeys(signer)},
        HostKeyCallback: ssh.InsecureIgnoreHostKey(),
        Timeout:         timeout,
    }
    c, err := ssh.Dial("tcp", net.JoinHostPort(addr, "22"), cfg)
    if err != nil {
        return nil, err
    }
    s, err := sftp.NewClient(c)
    if err != nil {
        _ = c.Close()
        return nil, err
    }
    return &SSHConn{client: c, sftp: s}, nil
}

func (s *SSHConn) Exec(ctx context.Context, cmd string, env map[string]string, sudo bool) (string, string, int, error) {
    sess, err := s.client.NewSession()
    if err != nil {
        return "", "", -1, err
    }
    defer sess.Close()
    for k, v := range env {
        if err := sess.Setenv(k, v); err != nil {
            return "", "", -1, err
        }
    }
    if sudo {
        cmd = fmt.Sprintf("sudo -n bash -lc %q", cmd)
    }
    var stdout, stderr io.Reader
    stdout, err = sess.StdoutPipe()
    if err != nil {
        return "", "", -1, err
    }
    stderr, err = sess.StderrPipe()
    if err != nil {
        return "", "", -1, err
    }
    if err := sess.Start(cmd); err != nil {
        return "", "", -1, err
    }
    done := make(chan error, 1)
    go func() { done <- sess.Wait() }()
    select {
    case <-ctx.Done():
        _ = sess.Signal(ssh.SIGKILL)
        return "", "", -1, ctx.Err()
    case err := <-done:
        bOut, _ := io.ReadAll(stdout)
        bErr, _ := io.ReadAll(stderr)
        exit := 0
        if err != nil {
            if ee, ok := err.(*ssh.ExitError); ok {
                exit = ee.ExitStatus()
            } else {
                exit = 1
            }
        }
        return string(bOut), string(bErr), exit, nil
    }
}

func (s *SSHConn) Put(ctx context.Context, src io.Reader, dst string, mode os.FileMode) error {
    f, err := s.sftp.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC)
    if err != nil {
        return err
    }
    defer f.Close()
    if _, err := io.Copy(f, src); err != nil {
        return err
    }
    return s.sftp.Chmod(dst, mode)
}

func (s *SSHConn) Get(ctx context.Context, src string) (io.ReadCloser, error) {
    return s.sftp.Open(src)
}

func (s *SSHConn) Close() error {
    _ = s.sftp.Close()
    return s.client.Close()
}

func osCreateTrunc() int {
    return 0x0001 | 0x0200
}
