package sportsmatrix

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"strconv"
	"strings"
	"syscall"

	"go.uber.org/zap"
)

func (s *SportsMatrix) launchWebBoard(ctx context.Context) error {
	args := []string{
		"--kiosk",
		fmt.Sprintf("--app=http://localhost:%d/board", s.cfg.HTTPListenPort),
	}
	cmd := exec.CommandContext(ctx, "/usr/bin/chromium-browser", args...)

	cmd.Env = os.Environ()

	// Chromium doesn't like to run as root, so we'll run it as someone else
	u, err := user.Lookup(s.cfg.WebBoardUser)
	if err != nil {
		return err
	}

	uid, err := strconv.Atoi(u.Uid)
	if err != nil {
		return err
	}

	gid, err := strconv.Atoi(u.Gid)
	if err != nil {
		return err
	}

	cmd.Env = append(cmd.Env, "DISPLAY=:0")
	cmd.Env = append(cmd.Env, fmt.Sprintf("HOME=%s", u.HomeDir))
	cmd.Env = append(cmd.Env, fmt.Sprintf("XAUTHORITY=%s/.Xauthority", u.HomeDir))

	cmd.SysProcAttr = &syscall.SysProcAttr{
		Credential: &syscall.Credential{
			// This should be "nobody"
			Uid: uint32(uid),
			Gid: uint32(gid),
		},
	}

	s.log.Info("launching web board to chromium browser",
		zap.String("command", strings.Join(cmd.Args, " ")),
		zap.String("user", s.cfg.WebBoardUser),
		zap.Int("UID", uid),
		zap.Int("GID", gid),
	)

	out, err := cmd.CombinedOutput()
	if err != nil {
		s.log.Error("exec error", zap.ByteString("error", out))
		return err
	}

	s.log.Error("exec output", zap.ByteString("output", out))

	return nil
}
