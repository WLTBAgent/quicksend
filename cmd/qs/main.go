package main

import (
	"fmt"
	"os"
	"strconv"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/wltbagent/quicksend/internal/client"
	"github.com/wltbagent/quicksend/internal/config"
)

var rootCmd = &cobra.Command{
	Use:   "qs",
	Short: "Quicksend CLI client",
}

var addKeyCmd = &cobra.Command{
	Use:   "add-key <nickname> <url> <key>",
	Short: "Associate a peer key with a nickname",
	Args:  cobra.ExactArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		nickname := args[0]
		url := args[1]
		key := args[2]

		if len(key) != 32 {
			return fmt.Errorf("key must be exactly 32 characters, got %d", len(key))
		}

		cfg, err := config.Load()
		if err != nil {
			return err
		}
		cfg.AddPeer(nickname, url, key)
		if err := cfg.Save(); err != nil {
			return err
		}
		fmt.Printf("added peer %q\n", nickname)
		return nil
	},
}

var rmKeyCmd = &cobra.Command{
	Use:   "rm-key <nickname>",
	Short: "Remove a peer key association",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		nickname := args[0]

		cfg, err := config.Load()
		if err != nil {
			return err
		}
		if !cfg.RemovePeer(nickname) {
			return fmt.Errorf("peer %q not found", nickname)
		}
		if err := cfg.Save(); err != nil {
			return err
		}
		fmt.Printf("removed peer %q\n", nickname)
		return nil
	},
}

var listPeersCmd = &cobra.Command{
	Use:   "peers",
	Short: "List all configured peers",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}
		if len(cfg.Peers) == 0 {
			fmt.Println("no peers configured. Use 'qs add-key <nickname> <url> <key>' to add one.")
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "NICKNAME\tURL\tKEY")
		for name, peer := range cfg.Peers {
			fmt.Fprintf(w, "%s\t%s\t%s...%s\n", name, peer.URL, peer.Key[:4], peer.Key[28:])
		}
		w.Flush()
		return nil
	},
}

var listCmd = &cobra.Command{
	Use:   "list <nickname>",
	Short: "List available files from a peer",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := peerClient(args[0])
		if err != nil {
			return err
		}

		files, err := c.List()
		if err != nil {
			return err
		}
		if len(files) == 0 {
			fmt.Println("no files available")
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "#\tNAME\tSIZE\tUPLOADED")
		for i, f := range files {
			fmt.Fprintf(w, "%d\t%s\t%s\t%s\n", i+1, f.Name, humanSize(f.Size), f.UploadedAt.Format("2006-01-02 15:04"))
		}
		w.Flush()
		return nil
	},
}

var downloadCmd = &cobra.Command{
	Use:   "download <nickname> <index>",
	Short: "Download a file by its index number",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := peerClient(args[0])
		if err != nil {
			return err
		}

		idx, err := strconv.Atoi(args[1])
		if err != nil || idx < 1 {
			return fmt.Errorf("index must be a positive number")
		}

		files, err := c.List()
		if err != nil {
			return err
		}
		if idx > len(files) {
			return fmt.Errorf("index %d out of range (1-%d)", idx, len(files))
		}

		file := files[idx-1]
		fmt.Printf("downloading %s...\n", file.Name)
		if err := c.Download(file.Name, ""); err != nil {
			return err
		}
		fmt.Printf("saved %s (%s)\n", file.Name, humanSize(file.Size))
		return nil
	},
}

var uploadCmd = &cobra.Command{
	Use:   "upload <nickname> <file>",
	Short: "Upload a file to a peer",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := peerClient(args[0])
		if err != nil {
			return err
		}

		filePath := args[1]
		if _, err := os.Stat(filePath); err != nil {
			return fmt.Errorf("file not found: %s", filePath)
		}

		fmt.Printf("uploading %s...\n", filePath)
		if err := c.Upload(filePath); err != nil {
			return err
		}
		fmt.Println("upload complete")
		return nil
	},
}

func peerClient(nickname string) (*client.Client, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}
	peer, ok := cfg.GetPeer(nickname)
	if !ok {
		return nil, fmt.Errorf("peer %q not found. Use 'qs add-key' to add it first", nickname)
	}
	return client.New(peer.URL, peer.Key), nil
}

func humanSize(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(b)/float64(div), "KMGTPE"[exp])
}

func init() {
	rootCmd.AddCommand(addKeyCmd)
	rootCmd.AddCommand(rmKeyCmd)
	rootCmd.AddCommand(listPeersCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(downloadCmd)
	rootCmd.AddCommand(uploadCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
