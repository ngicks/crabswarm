package commands

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	chatcli "github.com/ngicks/crabswarm/crabswarm/chat/cli"
)

func chatFollowCmd(parent *cobra.Command, flags *chatFlags) {
	var (
		flagAdmin  bool
		flagRoom   string
		flagFormat string
	)

	cmd := &cobra.Command{
		Use:   "follow [--admin [--room <room>]]",
		Short: "Stream the room's messages to stdout as JSON Lines",
		Long: `follow prints the messages of a room as they are sent, for another program
to read. It runs until it is stopped, and an interrupt ends it with status 0.

stdout carries one JSON object per line, UTF-8 without a byte order mark, each
ending in a newline and flushed as soon as it is written. stderr is free-form
logging and carries no part of the protocol. Every line has a "type":

  {"type":"status","message":"following <room>"}
      The stream opened on <room>.
  {"type":"status","message":"reconnecting to the daemon"}
      The stream was lost. It is written once per outage, however many attempts
      the outage takes, and the next "following" line ends it.
  {"type":"message","message":"<team>/<name>: <text>","inject":true,...}
      A message. "message" is the line to show, "inject" whether the message is
      for the follower: it names the follower or everyone, and the follower did
      not send it. The rest is the message as the schema names it: "id", "seq"
      (a number), "from", "target", "text", "sent_at", and "mentioned_you",
      which repeats "inject".

Following is not attendance. It puts nobody in the room, nudges nobody, and
moves no read position, so a token nobody attends under follows as well as one
that does. The stream starts live: it prints the messages sent after it opened.

A stream lost to the daemon going away is opened again after a pause that
starts at a second and doubles up to thirty, and it resumes after the last
message printed, so every message is printed once. A refused token or identity
ends the command with the refusal.

A reader that shows a message to its agent can mark it read with
` + "`chat read --skip SEQ`" + `. The read position is one sequence number per room,
so that marks every earlier unread mention read as well, shown or not.

The token is resolved as for every member verb. --admin follows as the host
operator instead, authenticated by --identity, and is mentioned by nothing: its
"inject" is always false. --room names the room it follows. Without --room it
follows the room of the current directory, the nearest of that directory and
its ancestors the daemon lists, and a directory under no room is followed
itself, since the agents that make it a room may start there later.`,
		Example: `  crabswarm chat follow
  crabswarm chat follow --admin --identity ~/.config/crabswarm/chat_admin.key
  crabswarm chat follow --admin --room /work/proj`,
		Args:              cobra.NoArgs,
		ValidArgsFunction: cobra.NoFileCompletions,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runChatFollow(cmd, args, flags, flagAdmin, flagRoom, flagFormat)
		},
	}

	f := cmd.Flags()
	f.BoolVar(&flagAdmin, "admin", false,
		"follow as the host operator, authenticated by --identity")
	f.StringVar(&flagRoom, "room", "",
		"with --admin, the room to follow; the room of the current directory when unset")
	f.StringVar(&flagFormat, "format", "jsonl", "output format; jsonl is the only one")
	// Hidden while jsonl is the only format: it is there so a reader can pin the
	// format it parses before a second one exists.
	_ = f.MarkHidden("format")

	parent.AddCommand(cmd)
}

func runChatFollow(
	cmd *cobra.Command,
	_ []string,
	flags *chatFlags,
	admin bool,
	room, format string,
) error {
	if format != "jsonl" {
		return fmt.Errorf("--format %q is not supported: jsonl is the only format", format)
	}
	if admin {
		return runChatFollowAdmin(cmd, flags, room)
	}
	if cmd.Flags().Changed("room") {
		return errors.New("--room names the room an --admin follow streams; " +
			"a member follows the room its token stands for")
	}
	client, token, err := dialChatAsMember(cmd, flags)
	if err != nil {
		return err
	}
	defer client.Close()

	return chatcli.FollowInto(cmd.Context(), commandLogger(cmd), cmd.OutOrStdout(),
		func(ctx context.Context, since *int64) (chatcli.FollowStream, error) {
			return client.Follow(ctx, token, since)
		})
}

func runChatFollowAdmin(cmd *cobra.Command, flags *chatFlags, room string) error {
	identity, err := chatIdentityPath(flags)
	if err != nil {
		return err
	}
	client, err := dialChat(cmd, flags)
	if err != nil {
		return err
	}
	defer client.Close()

	admin := client.Admin(identity)
	room, err = chatcli.RoomToFollow(cmd.Context(), admin, room, os.Getwd)
	if err != nil {
		return err
	}
	return chatcli.FollowInto(cmd.Context(), commandLogger(cmd), cmd.OutOrStdout(),
		func(ctx context.Context, since *int64) (chatcli.FollowStream, error) {
			return admin.Follow(ctx, room, since)
		})
}
