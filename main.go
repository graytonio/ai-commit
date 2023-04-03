package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"

	"github.com/sashabaranov/go-openai"
	"github.com/sirupsen/logrus"
)

const GPTPromptTemplate = `Given a git diff between two versions of a codebase, generate a concise, informative, and accurate commit message that summarizes the changes made in the diff. The commit message should follow good commit message guidelines, including starting with a capitalized verb in the imperative tense and providing a brief description of the changes made.

For example, if the git diff shows that a function was added to a file, the generated commit message could be "Add new function to improve performance." If the git diff shows that a line was deleted, the generated commit message could be "Remove unused code to simplify implementation." The goal is to generate a commit message that is useful for other developers to understand the changes made without having to look at the diff themselves.

%s`

var client *openai.Client

var (
    ErrNoStagedChanged = errors.New("no staged changes")
)

var msgPrefix string
var verbose bool

func init() {
    token, set := os.LookupEnv("OPENAI_TOKEN")
    if !set {
        logrus.Fatalln("OPENAI_TOKEN ENV not set")
    }
    client = openai.NewClient(token)

    flag.StringVar(&msgPrefix, "prefix", "", "prefix for commit message")
    flag.BoolVar(&verbose, "verbose", false, "verbose logging")
    flag.Parse()
}

func main() {
    if verbose {
        logrus.SetLevel(logrus.DebugLevel)
    }

    diff, err := getGitDiffString()
    if err != nil {
       logrus.Fatalln(err) 
    }

    logrus.Debugf("Git Diff: %s\n", diff)
    logrus.Debugln("Fetching AI Commit Message")

    commitMsg, err := generateCommitMessage(diff)
    if err != nil {
        logrus.Fatalln(err)
    }

    logrus.Debugf("AI Commit Prompt: %s\n", commitMsg)

    fullMsg := fmt.Sprintf("%s%s", msgPrefix, commitMsg)
    fmt.Printf("Making Commit: %s\n", fullMsg)
    createGitCommit(fullMsg)
}


func createPromptText(diff string) string {
    return fmt.Sprintf(GPTPromptTemplate, diff)
}

func generateCommitMessage(diff string) (string, error) {
    resp, err := client.CreateChatCompletion(context.Background(), openai.ChatCompletionRequest{
        Model: openai.GPT3Dot5Turbo0301,
        Messages: []openai.ChatCompletionMessage{
            {
                Role: openai.ChatMessageRoleUser,
                Content: createPromptText(diff),
            },
        },
   })

   if err != nil {
       return "", err
   }

   return resp.Choices[0].Message.Content, nil
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
