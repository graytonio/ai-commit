package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/cbess/go-textwrap"
	"github.com/sashabaranov/go-openai"
	"github.com/sirupsen/logrus"
)

const GPTSystemPrompt = `Given a git diff between two versions of a codebase and a message summarizing the previously processed changes, generate a concise, informative, and accurate commit message that summarizes the changes made in the diff. The commit message should follow good commit message guidelines, including starting with a capitalized verb in the imperative tense and providing a brief description of the changes made.

For example, if the git diff shows that a function was added to a file, the generated commit message could be "Add new function to improve performance." If the git diff shows that a line was deleted, the generated commit message could be "Remove unused code to simplify implementation." The goal is to generate a commit message that is useful for other developers to understand the changes made without having to look at the diff themselves.

The final commit message should be at most 50 characters.
`

const GPTUserMessageTemplate = `%s

%s`

var client *openai.Client

var (
	ErrNoStagedChanged = errors.New("no staged changes")
)

var msgPrefix string
var dryRun bool
var verbose bool

func init() {
	token, set := os.LookupEnv("OPENAI_TOKEN")
	if !set {
		logrus.Fatalln("OPENAI_TOKEN ENV not set")
	}
	client = openai.NewClient(token)

	flag.StringVar(&msgPrefix, "prefix", "", "prefix for commit message")
	flag.BoolVar(&verbose, "verbose", false, "verbose logging")
	flag.BoolVar(&dryRun, "dry", false, "process diff without making commit")
	flag.Parse()
}

func main() {
	if verbose {
		logrus.SetLevel(logrus.DebugLevel)
	}

    fmt.Println("Fetching Git Diff")

	diff, err := getGitDiffString()
	if err != nil {
        logrus.Fatalf("Error fetching git diff: %v", err)
	}

	logrus.Debugf("Git Diff: %s\n", diff)
	fmt.Println("Fetching AI Commit Message")

	commitMsg, err := generateCommitMessage(diff)
	if err != nil {
        logrus.Fatalf("Error generating commit message: %v", err)
	}

	logrus.Debugf("AI Commit Prompt: %s\n", commitMsg)

	fullMsg := fmt.Sprintf("%s%s", msgPrefix, commitMsg)

    fmt.Printf("Commit Message: %s\n", fullMsg)
    
    if dryRun {
        fmt.Println("Dry run enabled skipping commit")
        return
    }
	
    fmt.Println("Creating git commit")
	err = createGitCommit(fullMsg)
    if err != nil {
        logrus.Fatalf("Error creating commit: %v", err)
    }
}

func generateCommitMessage(diff string) (string, error) {
	chunks, err := textwrap.WordWrap(diff, 3500, -1)
	if err != nil {
		return "", err
	}

    ctx, cancel := context.WithTimeout(context.Background(), time.Second * 60)
    defer cancel()

    var rollingCommitSummary string
	for i, chunk := range chunks.TextGroups { 
	    var messages = []openai.ChatCompletionMessage{
		    {
			    Role:    openai.ChatMessageRoleSystem,
			    Content: GPTSystemPrompt,
		    },
            {
                Role: openai.ChatMessageRoleUser,
                Content: fmt.Sprintf(GPTUserMessageTemplate, rollingCommitSummary, chunk),
            },
	    }

	    logrus.WithField("iteration", i).WithField("state", "pre-completion").WithField("rollingCommitSummary", rollingCommitSummary).Debug()
    	
        resp, err := client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
	    	Model:    openai.GPT3Dot5Turbo0301,
		    Messages: messages,
	    })

        if err != nil {
            if errors.Is(err, context.DeadlineExceeded) {
                logrus.Warn("Context deadline exceeded returning current processed commit message")
                break // If deadline exceeded return the current commit message buffer
            }
            return "", err
        }

        rollingCommitSummary = resp.Choices[0].Message.Content
	    logrus.WithField("iteration", i).WithField("state", "post-completion").WithField("rollingCommitSummary", rollingCommitSummary).Debug()
    }

	return rollingCommitSummary, nil
}

func getGitDiffString() (string, error) {
	cmd := exec.Command("git", "diff", "--staged")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}

	// No diff
	if len(out) == 0 {
		return "", ErrNoStagedChanged
	}

	return string(out), nil
}

func createGitCommit(commit string) error {
	cmd := exec.Command("git", "commit", "-m", commit)
	_, err := cmd.Output()
	if err != nil {
		return err
	}

	return nil
}
