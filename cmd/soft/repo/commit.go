package repo

import (
	"fmt"
	"strings"
	"time"

	gansi "github.com/charmbracelet/glamour/ansi"
	"github.com/charmbracelet/soft-serve/cmd"
	"github.com/charmbracelet/soft-serve/git"
	"github.com/charmbracelet/soft-serve/pkg/access"
	"github.com/charmbracelet/soft-serve/pkg/backend"
	"github.com/charmbracelet/soft-serve/pkg/proto"
	"github.com/charmbracelet/soft-serve/pkg/ui/common"
	"github.com/charmbracelet/soft-serve/pkg/ui/styles"
	"github.com/muesli/termenv"
	"github.com/spf13/cobra"
)

// commitCommand returns a command that prints the contents of a commit.
func commitCommand() *cobra.Command {
	var color bool
	var patchOnly bool

	cmd := &cobra.Command{
		Use:   "commit SHA",
		Short: "Print out the contents of a diff",
		Args:  cobra.ExactArgs(2),
		RunE: func(co *cobra.Command, args []string) error {
			ctx := co.Context()
			be := backend.FromContext(ctx)
			repoName := args[0]
			commitSHA := args[1]

			rr, err := be.Repository(ctx, repoName)
			if err != nil {
				return err
			}

			if !cmd.CheckUserHasAccess(co, rr.Name(), access.ReadWriteAccess) {
				return proto.ErrUnauthorized
			}

			r, err := rr.Open()
			if err != nil {
				return err
			}

			commit, err := r.CommitByRevision(commitSHA)
			if err != nil {
				return err
			}

			patch, err := r.Patch(commit)
			if err != nil {
				return err
			}

			diff, err := r.Diff(commit)
			if err != nil {
				return err
			}

			commonStyle := styles.DefaultStyles()
			style := commonStyle.Log

			s := strings.Builder{}
			commitLine := "commit " + commitSHA
			authorLine := "Author: " + commit.Author.Name
			dateLine := "Date:   " + commit.Committer.When.UTC().Format(time.UnixDate)
			msgLine := strings.ReplaceAll(commit.Message, "\r\n", "\n")
			statsLine := renderStats(diff, commonStyle, color)
			diffLine := renderDiff(patch, color)

			if patchOnly {
				co.Println(
					diffLine,
				)
				return nil
			}

			if color {
				s.WriteString(fmt.Sprintf("%s\n%s\n%s\n%s\n",
					style.CommitHash.Render(commitLine),
					style.CommitAuthor.Render(authorLine),
					style.CommitDate.Render(dateLine),
					style.CommitBody.Render(msgLine),
				))
			} else {
				s.WriteString(fmt.Sprintf("%s\n%s\n%s\n%s\n",
					commitLine,
					authorLine,
					dateLine,
					msgLine,
				))
			}

			s.WriteString(fmt.Sprintf("\n%s\n%s",
				statsLine,
				diffLine,
			))

			co.Println(
				s.String(),
			)

			return nil
		},
	}

	cmd.Flags().BoolVarP(&color, "color", "c", false, "Colorize output")
	cmd.Flags().BoolVarP(&patchOnly, "patch", "p", false, "Output patch only")

	return cmd
}

func renderCtx() gansi.RenderContext {
	return gansi.NewRenderContext(gansi.Options{
		ColorProfile: termenv.TrueColor,
		Styles:       common.StyleConfig(),
	})
}

func renderDiff(patch string, color bool) string {
	c := patch

	if color {
		var s strings.Builder
		var pr strings.Builder

		diffChroma := &gansi.CodeBlockElement{
			Code:     patch,
			Language: "diff",
		}

		err := diffChroma.Render(&pr, renderCtx())

		if err != nil {
			s.WriteString(fmt.Sprintf("\n%s", err.Error()))
		} else {
			s.WriteString(fmt.Sprintf("\n%s", pr.String()))
		}

		c = s.String()
	}

	return c
}

func renderStats(diff *git.Diff, commonStyle *styles.Styles, color bool) string {
	style := commonStyle.Log
	c := diff.Stats().String()

	if color {
		s := strings.Split(c, "\n")

		for i, line := range s {
			ch := strings.Split(line, "|")
			if len(ch) > 1 {
				adddel := ch[len(ch)-1]
				adddel = strings.ReplaceAll(adddel, "+", style.CommitStatsAdd.Render("+"))
				adddel = strings.ReplaceAll(adddel, "-", style.CommitStatsDel.Render("-"))
				s[i] = strings.Join(ch[:len(ch)-1], "|") + "|" + adddel
			}
		}

		return strings.Join(s, "\n")
	}

	return c
}
