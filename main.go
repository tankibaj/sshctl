package main

import (
	"bufio"
	"crypto/rsa"
	"fmt"
	"io/fs"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"golang.org/x/crypto/ssh"
)

const (
	ColorReset  = "\033[0m"
	ColorRed    = "\033[31m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
	ColorBlue   = "\033[34m"
	ColorPurple = "\033[35m"
	ColorCyan   = "\033[36m"
	ColorWhite  = "\033[37m"
)

// MenuModel represents a selectable menu using bubbletea
type MenuModel struct {
	title     string
	items     []string
	cursor    int
	selected  int
	cancelled bool
	done      bool
}

type MenuResult struct {
	Index     int
	Cancelled bool
}

func NewMenuModel(title string, items []string) MenuModel {
	return MenuModel{
		title:     title,
		items:     items,
		cursor:    0,
		selected:  -1,
		cancelled: false,
		done:      false,
	}
}

func (m MenuModel) Init() tea.Cmd {
	return nil
}

func (m MenuModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			m.cancelled = true
			m.done = true
			return m, tea.Quit
		case "enter":
			m.selected = m.cursor
			m.done = true
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.items)-1 {
				m.cursor++
			}
		}
	}
	return m, nil
}

func (m MenuModel) View() string {
	s := fmt.Sprintf("%s%s%s\n", ColorCyan, m.title, ColorReset)
	s += fmt.Sprintf("%sUse ↑↓ (or j/k) to navigate, Enter to select, Esc/q to cancel%s\n\n", ColorYellow, ColorReset)

	for i, item := range m.items {
		cursor := "  "
		if m.cursor == i {
			cursor = fmt.Sprintf("%s❯ %s", ColorGreen, ColorReset)
		}
		s += fmt.Sprintf("%s%s\n", cursor, item)
	}

	s += fmt.Sprintf("\n%sPress Ctrl+C to force quit%s", ColorRed, ColorReset)
	return s
}

type SSHManager struct {
	sshDir     string
	configPath string
}

func NewSSHManager() *SSHManager {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatal("Could not get user home directory:", err)
	}

	sshDir := filepath.Join(homeDir, ".ssh")
	return &SSHManager{
		sshDir:     sshDir,
		configPath: filepath.Join(sshDir, "config"),
	}
}

func (sm *SSHManager) ensureSSHDir() error {
	return os.MkdirAll(sm.sshDir, 0700)
}

func (sm *SSHManager) copyPublicKey() error {
	pubKeys, err := sm.findPublicKeys()
	if err != nil {
		return err
	}

	if len(pubKeys) == 0 {
		fmt.Println("No public keys found in ~/.ssh/")
		return nil
	}

	selection, err := sm.showKeyMenu(pubKeys, "Select SSH public key to copy:")
	if err != nil {
		return err
	}

	if selection == -1 {
		fmt.Println("Selection cancelled")
		return nil
	}

	selectedKey := pubKeys[selection]
	content, err := ioutil.ReadFile(selectedKey)
	if err != nil {
		return fmt.Errorf("error reading key file: %v", err)
	}

	err = sm.copyToClipboard(string(content))
	if err != nil {
		return fmt.Errorf("error copying to clipboard: %v", err)
	}

	fmt.Printf(ColorGreen+"%s has been copied to clipboard\n"+ColorReset, filepath.Base(selectedKey))
	return nil
}

func (sm *SSHManager) copyToClipboard(text string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("pbcopy")
	case "linux":
		if _, err := exec.LookPath("xclip"); err == nil {
			cmd = exec.Command("xclip", "-selection", "clipboard")
		} else if _, err := exec.LookPath("xsel"); err == nil {
			cmd = exec.Command("xsel", "--clipboard", "--input")
		} else {
			return fmt.Errorf("no clipboard utility found (install xclip or xsel)")
		}
	case "windows":
		cmd = exec.Command("clip")
	default:
		return fmt.Errorf("unsupported platform")
	}

	cmd.Stdin = strings.NewReader(text)
	return cmd.Run()
}

func (sm *SSHManager) showKeyMenu(keys []string, title string) (int, error) {
	if len(keys) == 0 {
		return -1, fmt.Errorf("no keys to display")
	}

	// Check if we're in a terminal that supports interactive input
	if !isInteractiveTerminal() {
		// Fallback to numbered selection for non-interactive terminals
		return sm.showNumberedMenu(keys, title)
	}

	// Create simplified display names for the menu
	displayItems := make([]string, len(keys))
	for i, key := range keys {
		displayItems[i] = filepath.Base(key)
	}

	model := NewMenuModel(title, displayItems)
	p := tea.NewProgram(model)

	finalModel, err := p.Run()
	if err != nil {
		// Fallback to numbered selection if bubbletea fails
		return sm.showNumberedMenu(keys, title)
	}

	result := finalModel.(MenuModel)
	if result.cancelled || result.selected == -1 {
		return -1, nil
	}

	return result.selected, nil
}

func isInteractiveTerminal() bool {
	// Check if we're in an interactive terminal
	if os.Getenv("TERM") == "" {
		return false
	}

	// Check if stdin is a terminal
	fileInfo, _ := os.Stdin.Stat()
	return (fileInfo.Mode() & os.ModeCharDevice) != 0
}

func (sm *SSHManager) showNumberedMenu(keys []string, title string) (int, error) {
	fmt.Println(ColorCyan + title + ColorReset)
	for i, key := range keys {
		fmt.Printf("  %d) %s\n", i+1, filepath.Base(key))
	}

	var selection int
	for {
		fmt.Print(ColorYellow + "Pick a SSH Key (1-" + strconv.Itoa(len(keys)) + "): " + ColorReset)
		_, err := fmt.Scanln(&selection)
		if err != nil || selection < 1 || selection > len(keys) {
			fmt.Println(ColorRed + "Invalid selection. Please try again." + ColorReset)
			continue
		}
		break
	}

	return selection - 1, nil
}

func (sm *SSHManager) findPublicKeys() ([]string, error) {
	var pubKeys []string

	err := filepath.WalkDir(sm.sshDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.HasSuffix(path, ".pub") {
			pubKeys = append(pubKeys, path)
		}
		return nil
	})

	return pubKeys, err
}

func (sm *SSHManager) findPrivateKeys() ([]string, error) {
	var privKeys []string

	err := filepath.WalkDir(sm.sshDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && !strings.HasSuffix(path, ".pub") && !strings.Contains(path, "known_hosts") && !strings.Contains(path, "config") {
			// Check if it's actually a private key
			content, err := ioutil.ReadFile(path)
			if err == nil && strings.Contains(string(content), "PRIVATE") {
				privKeys = append(privKeys, path)
			}
		}
		return nil
	})

	return privKeys, err
}

func (sm *SSHManager) addAllKeys() error {
	privKeys, err := sm.findPrivateKeys()
	if err != nil {
		return err
	}

	if len(privKeys) == 0 {
		fmt.Println("No private keys found in ~/.ssh/")
		return nil
	}

	for _, key := range privKeys {
		var cmd *exec.Cmd
		if runtime.GOOS == "darwin" {
			cmd = exec.Command("ssh-add", "--apple-use-keychain", key)
		} else {
			cmd = exec.Command("ssh-add", key)
		}

		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err := cmd.Run()
		if err != nil {
			fmt.Printf(ColorRed+"Failed to add key %s: %v\n"+ColorReset, filepath.Base(key), err)
		} else {
			fmt.Printf(ColorGreen+"Added key: %s\n"+ColorReset, filepath.Base(key))
		}
	}

	return nil
}

func (sm *SSHManager) listAddedKeys() error {
	cmd := exec.Command("ssh-add", "-l")
	output, err := cmd.Output()
	if err != nil {
		fmt.Println("No keys currently added to ssh-agent")
		return nil
	}

	fmt.Println(ColorCyan + "Currently loaded SSH keys:" + ColorReset)
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		if line != "" {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				fingerprint := parts[1]
				// Try to match with local keys
				sm.matchLocalKey(fingerprint)
			}
		}
	}

	return nil
}

func (sm *SSHManager) matchLocalKey(fingerprint string) {
	pubKeys, _ := sm.findPublicKeys()
	for _, pubKey := range pubKeys {
		cmd := exec.Command("ssh-keygen", "-lf", pubKey)
		output, err := cmd.Output()
		if err != nil {
			continue
		}

		if strings.Contains(string(output), fingerprint) {
			fmt.Printf("  %s -> %s\n", fingerprint, pubKey)
			return
		}
	}
	fmt.Printf("  %s -> (unknown local key)\n", fingerprint)
}

func (sm *SSHManager) getAllKeys() error {
	pubKeys, err := sm.findPublicKeys()
	if err != nil {
		return err
	}

	if len(pubKeys) == 0 {
		fmt.Println("No public keys found in ~/.ssh/")
		return nil
	}

	fmt.Println(ColorCyan + "All SSH public keys:" + ColorReset)
	for _, pubKey := range pubKeys {
		cmd := exec.Command("ssh-keygen", "-lf", pubKey)
		output, err := cmd.Output()
		if err != nil {
			fmt.Printf("  %s (error reading)\n", pubKey)
			continue
		}

		parts := strings.Fields(string(output))
		if len(parts) >= 2 {
			fmt.Printf("  %s %s %s\n", parts[0], parts[1], pubKey)
		}
	}

	return nil
}

func (sm *SSHManager) openConfig() error {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		if runtime.GOOS == "windows" {
			editor = "notepad"
		} else {
			editor = "vim"
		}
	}

	cmd := exec.Command(editor, sm.configPath)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

func (sm *SSHManager) listHosts() error {
	if _, err := os.Stat(sm.configPath); os.IsNotExist(err) {
		fmt.Println("No SSH config file found")
		return nil
	}

	file, err := os.Open(sm.configPath)
	if err != nil {
		return err
	}
	defer file.Close()

	hosts := make(map[string]bool)
	scanner := bufio.NewScanner(file)

	fmt.Println(ColorCyan + "SSH config hosts:" + ColorReset)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "Host ") {
			host := strings.TrimPrefix(line, "Host ")
			host = strings.TrimSpace(host)
			if !hosts[host] && host != "*" {
				hosts[host] = true
				fmt.Printf("  %s\n", host)
			}
		}
	}

	return scanner.Err()
}

func (sm *SSHManager) removeHost(hostname string) error {
	if hostname == "" {
		return fmt.Errorf("hostname is required")
	}

	cmd := exec.Command("ssh-keygen", "-R", hostname)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to remove host: %v", err)
	}

	fmt.Printf(ColorGreen+"Removed %s from known_hosts\n"+ColorReset, hostname)
	return nil
}

// New functionality: Generate SSH key pair
func (sm *SSHManager) generateKey(keyName string, keyType string, bits int) error {
	if err := sm.ensureSSHDir(); err != nil {
		return err
	}

	// Interactive key name selection if not provided
	if keyName == "" {
		fmt.Print("Enter key name (default: id_rsa_new): ")
		fmt.Scanln(&keyName)
		if keyName == "" {
			keyName = "id_rsa_new"
		}
	}

	// Interactive key type selection if not provided or invalid
	if keyType == "" || (keyType != "rsa" && keyType != "ed25519") {
		keyTypes := []string{"ed25519 (recommended)", "rsa"}
		selection, err := sm.showKeyMenu(keyTypes, "Select key type:")
		if err != nil || selection == -1 {
			fmt.Println("Key generation cancelled")
			return nil
		}

		if selection == 0 {
			keyType = "ed25519"
		} else {
			keyType = "rsa"
		}
	}

	// Interactive bits selection for RSA if not provided
	if keyType == "rsa" && (bits != 2048 && bits != 4096) {
		bitOptions := []string{"4096 bits (recommended)", "2048 bits"}
		selection, err := sm.showKeyMenu(bitOptions, "Select key size:")
		if err != nil || selection == -1 {
			fmt.Println("Key generation cancelled")
			return nil
		}

		if selection == 0 {
			bits = 4096
		} else {
			bits = 2048
		}
	}

	privateKeyPath := filepath.Join(sm.sshDir, keyName)
	publicKeyPath := privateKeyPath + ".pub"

	// Check if key already exists
	if _, err := os.Stat(privateKeyPath); err == nil {
		fmt.Printf("Key %s already exists. Overwrite? (y/N): ", keyName)
		var response string
		fmt.Scanln(&response)
		if strings.ToLower(response) != "y" {
			fmt.Println("Operation cancelled")
			return nil
		}
	}

	var cmd *exec.Cmd
	switch keyType {
	case "rsa":
		cmd = exec.Command("ssh-keygen", "-t", "rsa", "-b", strconv.Itoa(bits), "-f", privateKeyPath, "-N", "")
	case "ed25519":
		cmd = exec.Command("ssh-keygen", "-t", "ed25519", "-f", privateKeyPath, "-N", "")
	default:
		cmd = exec.Command("ssh-keygen", "-t", "rsa", "-b", "4096", "-f", privateKeyPath, "-N", "")
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to generate key: %v", err)
	}

	fmt.Printf(ColorGreen + "Generated SSH key pair:\n" + ColorReset)
	fmt.Printf("  Private key: %s\n", privateKeyPath)
	fmt.Printf("  Public key:  %s\n", publicKeyPath)

	return nil
}

// New functionality: Test SSH connection
func (sm *SSHManager) testConnection(host string) error {
	if host == "" {
		return fmt.Errorf("hostname is required")
	}

	fmt.Printf("Testing SSH connection to %s...\n", host)

	cmd := exec.Command("ssh", "-o", "ConnectTimeout=10", "-o", "BatchMode=yes", host, "echo 'Connection successful'")
	output, err := cmd.Output()

	if err != nil {
		fmt.Printf(ColorRed+"Connection failed: %v\n"+ColorReset, err)
		return err
	}

	fmt.Printf(ColorGreen+"✓ %s"+ColorReset, string(output))
	return nil
}

// New functionality: Backup SSH keys
func (sm *SSHManager) backupKeys(backupPath string) error {
	if backupPath == "" {
		backupPath = fmt.Sprintf("ssh_backup_%s.tar.gz", time.Now().Format("20060102_150405"))
	}

	cmd := exec.Command("tar", "-czf", backupPath, "-C", filepath.Dir(sm.sshDir), ".ssh")
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("backup failed: %v", err)
	}

	fmt.Printf(ColorGreen+"SSH keys backed up to: %s\n"+ColorReset, backupPath)
	return nil
}

// New functionality: Key analysis
func (sm *SSHManager) analyzeKeys() error {
	pubKeys, err := sm.findPublicKeys()
	if err != nil {
		return err
	}

	fmt.Println(ColorCyan + "SSH Key Analysis:" + ColorReset)
	fmt.Println("─────────────────────────────────────────")

	for _, pubKey := range pubKeys {
		content, err := ioutil.ReadFile(pubKey)
		if err != nil {
			continue
		}

		// Parse the public key
		publicKey, _, _, _, err := ssh.ParseAuthorizedKey(content)
		if err != nil {
			fmt.Printf("  %s: %sCould not parse%s\n", filepath.Base(pubKey), ColorRed, ColorReset)
			continue
		}

		keyType := publicKey.Type()

		// Get file info for creation time
		info, _ := os.Stat(pubKey)

		fmt.Printf("  %s%s%s:\n", ColorYellow, filepath.Base(pubKey), ColorReset)
		fmt.Printf("    Type: %s\n", keyType)
		fmt.Printf("    Created: %s\n", info.ModTime().Format("2006-01-02 15:04:05"))

		// Analyze key strength
		switch keyType {
		case "ssh-rsa":
			if rsaKey, ok := publicKey.(ssh.CryptoPublicKey); ok {
				if rsaPubKey, ok := rsaKey.CryptoPublicKey().(*rsa.PublicKey); ok {
					bits := rsaPubKey.N.BitLen()
					fmt.Printf("    Bits: %d ", bits)
					if bits < 2048 {
						fmt.Printf("%s(WEAK)%s\n", ColorRed, ColorReset)
					} else if bits >= 4096 {
						fmt.Printf("%s(STRONG)%s\n", ColorGreen, ColorReset)
					} else {
						fmt.Printf("%s(GOOD)%s\n", ColorYellow, ColorReset)
					}
				}
			}
		case "ssh-ed25519":
			fmt.Printf("    %s(EXCELLENT - Modern curve)%s\n", ColorGreen, ColorReset)
		}
		fmt.Println()
	}

	return nil
}

func printHelp() {
	fmt.Printf("%sSSH Key Manager (sshctl)%s\n", ColorCyan, ColorReset)
	fmt.Println("═══════════════════════════════════════════════════════════════")
	fmt.Printf("%sBasic Operations:%s\n", ColorYellow, ColorReset)
	fmt.Println("  -a, --add, add                 Add all private keys to ssh-agent")
	fmt.Println("  -c, --copy, copy               Copy selected public key to clipboard")
	fmt.Println("  -C, --config, config           Open ~/.ssh/config file in editor")
	fmt.Println("  -g, --get, get                 List all keys from ~/.ssh/ directory")
	fmt.Println("  -h, --help, help               Display this help")
	fmt.Println("  -H, --host, host               List all hosts from ~/.ssh/config")
	fmt.Println("  -l, --list, list               List currently loaded keys in ssh-agent")
	fmt.Println("  -r, --remove, remove <host>    Remove host from ~/.ssh/known_hosts")
	fmt.Println()
	fmt.Printf("%sAdvanced Operations:%s\n", ColorYellow, ColorReset)
	fmt.Println("  -G, --generate, generate [name] [type] [bits]  Generate new SSH key pair")
	fmt.Println("       Types: rsa, ed25519 (default: rsa)")
	fmt.Println("       Bits: 2048, 4096 (default: 4096, RSA only)")
	fmt.Println("  -t, --test, test <host>        Test SSH connection to host")
	fmt.Println("  -b, --backup, backup [path]    Backup SSH directory")
	fmt.Println("  -A, --analyze, analyze         Analyze SSH keys for security")
	fmt.Println()
	fmt.Printf("%sExamples:%s\n", ColorYellow, ColorReset)
	fmt.Println("  sshctl -c                      # Interactive key selection for clipboard")
	fmt.Println("  sshctl -G mykey ed25519        # Generate Ed25519 key named 'mykey'")
	fmt.Println("  sshctl -t user@example.com     # Test connection to server")
	fmt.Println("  sshctl -r old-server.com       # Remove old server from known_hosts")
	fmt.Println("  sshctl -A                      # Analyze all keys for security issues")
}

func main() {
	if len(os.Args) < 2 {
		printHelp()
		return
	}

	sm := NewSSHManager()
	option := os.Args[1]

	switch option {
	case "-a", "--add", "add":
		if err := sm.addAllKeys(); err != nil {
			log.Fatal("Error adding keys:", err)
		}
	case "-c", "--copy", "copy":
		if err := sm.copyPublicKey(); err != nil {
			log.Fatal("Error copying key:", err)
		}
	case "-C", "--config", "config":
		if err := sm.openConfig(); err != nil {
			log.Fatal("Error opening config:", err)
		}
	case "-g", "--get", "get":
		if err := sm.getAllKeys(); err != nil {
			log.Fatal("Error getting keys:", err)
		}
	case "-H", "--host", "host":
		if err := sm.listHosts(); err != nil {
			log.Fatal("Error listing hosts:", err)
		}
	case "-h", "--help", "help":
		printHelp()
	case "-l", "--list", "list":
		if err := sm.listAddedKeys(); err != nil {
			log.Fatal("Error listing keys:", err)
		}
	case "-r", "--remove", "remove":
		if len(os.Args) < 3 {
			fmt.Println(ColorRed + "Usage: sshctl --remove <HOST IP | HOST NAME>" + ColorReset)
			os.Exit(1)
		}
		if err := sm.removeHost(os.Args[2]); err != nil {
			log.Fatal("Error removing host:", err)
		}
	case "-G", "--generate", "generate":
		keyName := ""
		keyType := "rsa"
		bits := 4096

		if len(os.Args) > 2 {
			keyName = os.Args[2]
		}
		if len(os.Args) > 3 {
			keyType = os.Args[3]
		}
		if len(os.Args) > 4 {
			if b, err := strconv.Atoi(os.Args[4]); err == nil {
				bits = b
			}
		}

		if err := sm.generateKey(keyName, keyType, bits); err != nil {
			log.Fatal("Error generating key:", err)
		}
	case "-t", "--test", "test":
		if len(os.Args) < 3 {
			fmt.Println(ColorRed + "Usage: sshctl --test <HOST>" + ColorReset)
			os.Exit(1)
		}
		if err := sm.testConnection(os.Args[2]); err != nil {
			log.Fatal("Error testing connection:", err)
		}
	case "-b", "--backup", "backup":
		backupPath := ""
		if len(os.Args) > 2 {
			backupPath = os.Args[2]
		}
		if err := sm.backupKeys(backupPath); err != nil {
			log.Fatal("Error backing up keys:", err)
		}
	case "-A", "--analyze", "analyze":
		if err := sm.analyzeKeys(); err != nil {
			log.Fatal("Error analyzing keys:", err)
		}
	default:
		fmt.Printf(ColorRed+"Unknown option: %s\n"+ColorReset, option)
		printHelp()
		os.Exit(1)
	}
}
