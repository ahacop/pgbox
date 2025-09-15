// Package render converts in-memory models to Docker artifact files
package render

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"
)

// AnchorMarker represents the start and end markers for an anchored block
type AnchorMarker struct {
	Start string // Start marker pattern
	End   string // End marker pattern
}

// DockerfileAnchors defines anchors for Dockerfile
var DockerfileAnchors = AnchorMarker{
	Start: "# pgbox: BEGIN",
	End:   "# pgbox: END",
}

// ComposeAnchors defines anchors for docker-compose.yml
var ComposeAnchors = AnchorMarker{
	Start: "# pgbox: BEGIN",
	End:   "# pgbox: END",
}

// ParsedFile represents a file with anchored regions identified
type ParsedFile struct {
	PreAnchor  []string // Lines before the anchored region
	Anchored   []string // Lines within the anchored region (will be replaced)
	PostAnchor []string // Lines after the anchored region
	HasAnchor  bool     // Whether an anchored region was found
}

// ParseFileWithAnchors parses a file and identifies anchored regions
func ParseFileWithAnchors(path string, marker AnchorMarker) (*ParsedFile, error) {
	// If file doesn't exist, return empty parsed file
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return &ParsedFile{
			PreAnchor:  []string{},
			Anchored:   []string{},
			PostAnchor: []string{},
			HasAnchor:  false,
		}, nil
	}

	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	parsed := &ParsedFile{
		PreAnchor:  []string{},
		Anchored:   []string{},
		PostAnchor: []string{},
		HasAnchor:  false,
	}

	scanner := bufio.NewScanner(file)
	inAnchor := false
	foundEnd := false

	for scanner.Scan() {
		line := scanner.Text()

		if !inAnchor && strings.Contains(line, marker.Start) {
			// Found start of anchor
			inAnchor = true
			parsed.HasAnchor = true
			continue // Don't include the marker itself
		}

		if inAnchor && strings.Contains(line, marker.End) {
			// Found end of anchor
			inAnchor = false
			foundEnd = true
			continue // Don't include the marker itself
		}

		// Add line to appropriate section
		if inAnchor {
			parsed.Anchored = append(parsed.Anchored, line)
		} else if !parsed.HasAnchor {
			parsed.PreAnchor = append(parsed.PreAnchor, line)
		} else if foundEnd {
			parsed.PostAnchor = append(parsed.PostAnchor, line)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading file: %w", err)
	}

	return parsed, nil
}

// ReplaceAnchored replaces the anchored section of a parsed file
func ReplaceAnchored(parsed *ParsedFile, marker AnchorMarker, newContent []string) []string {
	var result []string

	// Add pre-anchor content
	result = append(result, parsed.PreAnchor...)

	// Add anchored content with markers
	if len(newContent) > 0 || parsed.HasAnchor {
		result = append(result, marker.Start)
		result = append(result, newContent...)
		result = append(result, marker.End)
	}

	// Add post-anchor content
	result = append(result, parsed.PostAnchor...)

	return result
}

// WriteLines writes lines to a file
func WriteLines(path string, lines []string) error {
	content := strings.Join(lines, "\n")
	if len(lines) > 0 && !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	return os.WriteFile(path, []byte(content), 0644)
}

// ParseInitSQLAnchors parses init.sql with named anchor blocks
func ParseInitSQLAnchors(path string) (map[string][]string, []string, error) {
	blocks := make(map[string][]string)
	var preContent []string
	var currentBlock string
	var currentLines []string

	// If file doesn't exist, return empty
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return blocks, preContent, nil
	}

	file, err := os.Open(path)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Pattern for anchor markers
	startPattern := regexp.MustCompile(`^-- pgbox: begin (\S+)`)
	endPattern := regexp.MustCompile(`^-- pgbox: end (\S+)`)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()

		// Check for start marker
		if matches := startPattern.FindStringSubmatch(line); len(matches) > 1 {
			if currentBlock != "" {
				// Save previous block
				blocks[currentBlock] = currentLines
			}
			currentBlock = matches[1]
			currentLines = []string{}
			continue
		}

		// Check for end marker
		if matches := endPattern.FindStringSubmatch(line); len(matches) > 1 {
			if currentBlock == matches[1] {
				blocks[currentBlock] = currentLines
				currentBlock = ""
				currentLines = []string{}
			}
			continue
		}

		// Add line to current context
		if currentBlock != "" {
			currentLines = append(currentLines, line)
		} else if len(blocks) == 0 {
			// Before any blocks
			preContent = append(preContent, line)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, nil, fmt.Errorf("error reading file: %w", err)
	}

	return blocks, preContent, nil
}

// IndentLines indents lines by the specified number of spaces
func IndentLines(lines []string, spaces int) []string {
	indent := strings.Repeat(" ", spaces)
	result := make([]string, len(lines))
	for i, line := range lines {
		if line != "" {
			result[i] = indent + line
		} else {
			result[i] = line
		}
	}
	return result
}