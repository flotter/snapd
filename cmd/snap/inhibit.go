// -*- Mode: Go; indent-tabs-mode: t -*-

/*
 * Copyright (C) 2020 Canonical Ltd
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License version 3 as
 * published by the Free Software Foundation.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 *
 */

package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/canonical/x-go/i18n"
	"github.com/snapcore/snapd/cmd/snaplock/runinhibit"
	"github.com/snapcore/snapd/progress"
	"github.com/snapcore/snapd/usersession/client"
)

func waitWhileInhibited(snapName string) error {
	hint, err := runinhibit.IsLocked(snapName)
	if err != nil {
		return err
	}
	if hint == runinhibit.HintNotInhibited {
		return nil
	}

	// wait for HintInhibitedForRefresh set by gate-auto-refresh hook handler
	// when it has finished; the hook starts with HintInhibitedGateRefresh lock
	// and then either unlocks it or changes to HintInhibitedForRefresh (see
	// gateAutoRefreshHookHandler in hooks.go).
	// waitInhibitUnlock will return also on HintNotInhibited.
	notInhibited, err := waitInhibitUnlock(snapName, runinhibit.HintInhibitedForRefresh)
	if err != nil {
		return err
	}
	if notInhibited {
		return nil
	}

	if isGraphicalSession() {
		return graphicalSessionFlow(snapName, hint)
	}
	// terminal and headless
	return textFlow(snapName, hint)
}

func inhibitMessage(snapName string, hint runinhibit.Hint) string {
	switch hint {
	case runinhibit.HintInhibitedForRefresh:
		return fmt.Sprintf(i18n.G("snap package %q is being refreshed, please wait"), snapName)
	default:
		return fmt.Sprintf(i18n.G("snap package cannot be used now: %s"), string(hint))
	}
}

var isGraphicalSession = func() bool {
	return os.Getenv("DISPLAY") != "" || os.Getenv("WAYLAND_DISPLAY") != ""
}

var pendingRefreshNotification = func(refreshInfo *client.PendingSnapRefreshInfo) error {
	userclient := client.NewForUids(os.Getuid())
	if err := userclient.PendingRefreshNotification(context.TODO(), refreshInfo); err != nil {
		return err
	}
	return nil
}

var finishRefreshNotification = func(refreshInfo *client.FinishedSnapRefreshInfo) error {
	userclient := client.NewForUids(os.Getuid())
	if err := userclient.FinishRefreshNotification(context.TODO(), refreshInfo); err != nil {
		return err
	}
	return nil
}

func graphicalSessionFlow(snapName string, hint runinhibit.Hint) error {
	refreshInfo := client.PendingSnapRefreshInfo{
		InstanceName: snapName,
		// Remaining time = 0 results in "Snap .. is refreshing now" message from
		// usersession agent.
		TimeRemaining: 0,
	}

	if err := pendingRefreshNotification(&refreshInfo); err != nil {
		return err
	}
	if _, err := waitInhibitUnlock(snapName, runinhibit.HintNotInhibited); err != nil {
		return err
	}

	finishRefreshInfo := client.FinishedSnapRefreshInfo{InstanceName: snapName}
	return finishRefreshNotification(&finishRefreshInfo)
}

func textFlow(snapName string, hint runinhibit.Hint) error {
	fmt.Fprintf(Stdout, "%s\n", inhibitMessage(snapName, hint))
	pb := progress.MakeProgressBar()
	pb.Spin(i18n.G("please wait..."))
	_, err := waitInhibitUnlock(snapName, runinhibit.HintNotInhibited)
	pb.Finished()
	return err
}

var isLocked = runinhibit.IsLocked

// waitInhibitUnlock waits until the runinhibit lock hint has a specific waitFor value
// or isn't inhibited anymore.
var waitInhibitUnlock = func(snapName string, waitFor runinhibit.Hint) (notInhibited bool, err error) {
	// Every 0.5s check if the inhibition file is still present.
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			// Half a second has elapsed, let's check again.
			hint, err := isLocked(snapName)
			if err != nil {
				return false, err
			}
			if hint == runinhibit.HintNotInhibited {
				return true, nil
			}
			if hint == waitFor {
				return false, nil
			}
		}
	}
}
