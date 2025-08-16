// Copyright © 2019 Marcin Wojnarowski xmarcinmarcin@gmail.com
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/fatih/color"
	"github.com/kazec/driveignore/utils"
	"github.com/spf13/cobra"
)

// diffCmd represents the upload command
var diffCmd = &cobra.Command{
	Use:   "diff [drive sync folder path]",
	Short: "Compares your directory with the drive one",
	Long: `Prints out the difference in files between your source (input) and
drive sync folder ([drive sync folder path])

Red    - your drive sync folder is missing a file
Yellow - your drive sync folder has a file that doesnt exist in input
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		vPrint := utils.VPrintWrapper(verbose)
		redPrint := color.New(color.FgRed).PrintlnFunc()
		yellowPrint := color.New(color.FgHiYellow).PrintlnFunc()

		driveignore, driveignoreType := utils.DriveIgnore(diffInput, diffMergeIgnores)

		switch driveignoreType {
		case utils.GlobalIgnore:
			vPrint("loaded global .driveignore")
		case utils.LocalIgnore:
			vPrint("loaded local .driveignore")
		case utils.MergedIgnore:
			vPrint("loaded merged global and local .driveignore")
		case utils.NoIgnore:
			return errors.New("No local nor global .driveignores found")
		}

		missing, old := make(chan string), make(chan string)

		var err1, err2 error

		// search for missing files
		go func() {
			err1 = utils.Walker(diffInput, func(currPath string, info os.FileInfo, relativePath string) error {
				// ignore .driveignore files/dirs
				if info.IsDir() && driveignore.Match(currPath, true) {
					return filepath.SkipDir
				} else if !info.IsDir() && driveignore.Match(currPath, false) {
					return nil
				}

				// check if file/directory exists in drive sync folder
				goalPath := filepath.Join(args[0], relativePath)
				goalStat, err := os.Stat(goalPath)
				if os.IsNotExist(err) || (!os.SameFile(info, goalStat) && !info.IsDir()) {
					missing <- relativePath
				}
				return nil
			})
			close(missing)
		}()

		// search for legacy files/directories
		go func() {
			err2 = utils.Walker(args[0], func(currPath string, info os.FileInfo, relativePath string) error {
				// check if file exists in input folder
				inputPath := filepath.Join(diffInput, relativePath)
				goalStat, err := os.Stat(inputPath)
				if os.IsNotExist(err) || (!os.SameFile(info, goalStat) && !info.IsDir()) {
					old <- relativePath
				}
				return nil
			})
			close(old)
		}()

		for m := range missing {
			redPrint(m)
		}
		if err1 != nil {
			return err1
		}
		for o := range old {
			yellowPrint(o)
		}
		if err2 != nil {
			return err2
		}
		return nil
	},
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			return errors.New("There should only be one argument")
		}
		fstat, err := os.Stat(args[0])
		if os.IsNotExist(err) {
			return errors.New("Passed path doesnt exist")
		}
		if !fstat.IsDir() {
			return errors.New("Passed path isnt a directory")
		}
		return nil
	},
}

var diffInput string
var diffMergeIgnores bool

func init() {
	rootCmd.AddCommand(diffCmd)

	// Local flags
	diffCmd.Flags().StringVarP(&diffInput, "input", "i", ".", "Input directory of the files to be compared")
	diffCmd.Flags().BoolVarP(&diffMergeIgnores, "merge-ignores", "M", false, "Merges global and input dir .driveignore")
}
