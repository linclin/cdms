package actions

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/linclin/fastflow/pkg/entity/run"
	"github.com/melbahja/goph"
	"golang.org/x/crypto/ssh"
)

const (
	ActionKeySSH = "ssh"
)

type SSHParams struct {
	User    string `json:"user"`
	Ip      string `json:"ip"`
	Port    uint   `json:"port"`
	Key     string `json:"key"`
	Cmd     string `json:"cmd"`
	Timeout int    `json:"timeout"`
}

type SSH struct {
}

func (s *SSH) Name() string {
	return ActionKeySSH
}

// ParameterNew
func (s *SSH) ParameterNew() interface{} {
	return &SSHParams{}
}

// Run
func (s *SSH) Run(ctx run.ExecuteContext, params interface{}) (err error) {
	ctx.Trace("[Action ssh]start " + fmt.Sprintln("params", params))
	p, ok := params.(*SSHParams)
	if !ok {
		err = fmt.Errorf("[Action ssh]params type mismatch, want *SshParams, got %T", params)
		ctx.Trace(err.Error())
		return err
	}
	if p.User == "" || p.Ip == "" || p.Key == "" || p.Cmd == "" {
		err = fmt.Errorf("[Action ssh]SshParams user/ip/key/cmd cannot be empty")
		ctx.Trace(err.Error())
		return err
	}
	timeout := p.Timeout
	if timeout == 0 {
		timeout = 300
	}
	sshPort := p.Port
	if sshPort == 0 {
		sshPort = 22
	}
	sshAuth, err := goph.Key("./storage/ssh-key/"+p.Key, "")
	if err != nil {
		ctx.Trace("[Action ssh]sshAuth " + err.Error())
		return err
	}
	sshClient, err := goph.NewConn(&goph.Config{
		User:     p.User,
		Addr:     p.Ip,
		Port:     sshPort,
		Auth:     sshAuth,
		Callback: VerifyHost,
	})
	if err != nil {
		ctx.Trace("[Action ssh]NewConn " + err.Error())
		return err
	}
	defer sshClient.Close()
	context, cancel := context.WithTimeout(ctx.Context(), time.Duration(p.Timeout)*time.Second)
	defer cancel()
	out, err := sshClient.RunContext(context, p.Cmd)
	//out, err := sshClient.Run(p.Cmd)
	if err != nil {
		ctx.Trace("[Action ssh]Run " + err.Error())
		return err
	}
	ctx.Trace("[Action ssh]success " + fmt.Sprintln("params", params) + string(out))
	return nil
}

func VerifyHost(host string, remote net.Addr, key ssh.PublicKey) error {

	//
	// If you want to connect to new hosts.
	// here your should check new connections public keys
	// if the key not trusted you shuld return an error
	//
	// hostFound: is host in known hosts file.
	// err: error if key not in known hosts file OR host in known hosts file but key changed!
	hostFound, err := goph.CheckKnownHost(host, remote, key, "")
	// Host in known hosts but key mismatch!
	// Maybe because of MAN IN THE MIDDLE ATTACK!
	if hostFound && err != nil {
		return err
	}
	// handshake because public key already exists.
	if hostFound && err == nil {
		return nil
	}
	// Add the new host to known hosts file.
	return goph.AddKnownHost(host, remote, key, "")
}
