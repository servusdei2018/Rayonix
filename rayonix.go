/*
 * Rayonix
 * Copyright (C) 2019. All Rights Reserved.
 * A command line preprocessor toolchain for LibertyBasic and JustBasic.
 *
 * Features:
 *   -Parses in-file commented flags
 *   -Can manage multiple-files deployments (multiple files combine into
 *    one, which can be run with Liberty or JustBasic)
 *   -Such multiple-files deployments make sure there are no duplicate
 *    functions, subroutines, etc.
 * 
 * ToDo:
 *   -Must make "import folder/file.bas" which contains "import f2.bas"
 *    work -- f2.bas is in "folder/f2.bas" but context is of mainfile.
 *   -To make it work, foldername would have to be inherited from the
 *    including file along with filename. So two args to add to pFile:
 *    path and fname.
 *
 * Currently implemented:
 *   -"'!rayonix import relative/path/to/file.bas"
 *   -"'!rayonix import /absolute/path/to/file.bas"
 *   -"'!rayonix meta http://www.example.com/file.bas"
**/

package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
)

var (
	op string
	mainFile string
)

func init() {

	/*
	 * Get op(eration).
	 *
	 * Valid op(erations):
	 *  ./rayonix init <projectName> <mainFile.bas>
	 *     initliazes a new rayonix project.
	 *  ./rayonix build <mainFile.bas> <outFile.bas>
	 *     builds a rayonix project into a file executable via JustBasic
	 *     or LibertyBasic
	 *  ./rayonix doc
	 *     shows documentation
	 *  ./rayonix disclaimer
	 *     shows disclaimer
	**/

	if (len(os.Args) == 1 ) {
			displayUsage()
			return
	} else {

		op = os.Args[1]

		if (op!="init" && op!="build" && op!="doc" && op!="disclaimer"){
				displayUsage()
				return
		}
	}

	switch(op) {
		case "init":
			if len(os.Args) != 4 {
				displayUsage()
			}
			break
		case "build":
			if len(os.Args) != 4 {
				displayUsage()
			}
			break
		case "doc":
			displayDocumentation()
			break
		case "disclaimer":
			displayLicense()
	}
}

func main() {

	/*
	 * Main application entrypoint.
	**/

	switch(op) {
		case "init":
			initializeProject(os.Args[2], os.Args[3])
		case "build":
			mainFile = os.Args[2]
			buildProject(os.Args[2], os.Args[3])
	}
	
	return
}

func initializeProject(projectName string, projectFile string) {

	/*
	 * Initialize the new rayonix project.
	 *
	 * Creates two folders: "projectName"
	**/

	//Get working directory
	wd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	//Change to that directory
	err = os.Chdir(wd)
	if err != nil {
		log.Fatal(err)
	}
	//Make project directory
	err = os.Mkdir(projectName, os.FileMode(0777))
	if err != nil {
		log.Fatal(err)
	}
	//Change to that directory
	err = os.Chdir(projectName)
	if err != nil {
		log.Fatal(err)
	}
	//Make project file
	f, err := os.Create(projectFile)
	if err != nil {
		log.Fatal(err)
		f.Close()
	}
	//Write a simple file
	fmt.Fprint(f, "'My new Rayonix project\r\n")
	fmt.Fprint(f, "call main$\r\n\nsub main$\r\n")
	fmt.Fprint(f, "\tprint \"Hello, Rayonix!\"")
	fmt.Fprint(f, "end sub\r\n")
	err = f.Close()
	if err != nil {
		log.Fatal(err)
	}
}

func buildProject(filePath string, outPath string) {

	/*
	 * Build the Rayonix project.
	 *
	 * Parses:
	 *   '!rayonix import path/to/file.bas
	**/

	var finished []string //Finished document
	var data []string     //Input document
	var toAppend []string //Stuff to append to the finished document

	//Read main file
	bytes, ioErr := ioutil.ReadFile(filePath)
	if ioErr != nil {
		log.Fatal(ioErr)
	}

	//Check newline value
	if strings.Contains(string(bytes), "\r\n") {
		data = strings.Split(string(bytes), "\r\n")
	} else {
		data = strings.Split(string(bytes), "\n")
	}

	//Parse each line for a !rayonix command
	for lnIndex:=0; lnIndex < len(data); lnIndex++ {
			oline := data[lnIndex]
			line := strings.Replace(oline, "	", " ", -1)

			//Get rid of all pre-statements spaces
			for strings.HasPrefix(line, " ") {
					line = strings.TrimPrefix(line, " ")
			}

			if strings.HasPrefix(line, "'!rayonix") {
					//Parse this
					toAppend = process(line+"\r\n", toAppend, filePath)
			} else {
					//Add this to the finished product
					finished = append(finished, oline+"\r\n")
			}
	}

	//Append toAppend to finished
	for lnIndex:=0; lnIndex < len(toAppend); lnIndex++ {
			line := toAppend[lnIndex]
			finished = append(finished, line) //rm +"\r\n"
	}

	//Create the output
	f, err := os.Create(outPath)
	if err != nil {
		log.Fatal(err)
		f.Close()
	}

	//Write output
	for lnIndex:=0; lnIndex < len(finished); lnIndex++ {
			line := finished[lnIndex]
			fmt.Fprintln(f, line)
	}
	err = f.Close()
	if err != nil {
		log.Fatal(err)
	}
}

func pFile(filePath string, toAppend []string, context string) []string {

	/*
	 * Parses a file (which may include !rayonix commands
	 * which was included by another file.
	**/
		
	var data []string      //Input document
	var subAppend []string //Sub append document

	//Open and read from file
	bytes, ioErr := ioutil.ReadFile(filePath)
	if ioErr != nil {
		oldIoErr := ioErr
		//Absolute path tried, and failed. Now, try a relative one --
		//relative to the mainfile.
		fps := strings.Split(mainFile, "/")
		var newfps string
		for i:=0; i<len(fps)-1; i++ { 
				//Get the folder containing the mainfile
				newfps += fps[i] + "/"
		}
		
		//Append the relative path to that
		newfps += filePath
		
		bytes, ioErr = ioutil.ReadFile(newfps)
		
		if ioErr != nil {
		
			//Finally, last resort: try a filepath relative to the
			//caller.
	
			oldIoErr2 := ioErr
			
			fps2 := strings.Split(context, "/")
			var newfps2 string
			
			for i:=0; i<len(fps2)-1; i++ { 
				newfps2 += fps2[i] + "/"
			}
			newfps2 += filePath
			bytes, ioErr = ioutil.ReadFile(newfps2)
		
			if ioErr != nil {
					log.Println(oldIoErr)
					log.Println(oldIoErr2)
					log.Fatal(ioErr)
			}	
		}
	}

	//Detect newline character
	if strings.Contains(string(bytes), "\r\n") {
		data = strings.Split(string(bytes), "\r\n")
	} else {
		data = strings.Split(string(bytes), "\n")
	}

	//Loop through lines looking for !rayonix commands
	for lnIndex:=0; lnIndex < len(data); lnIndex++ {
			oline := data[lnIndex]
			line := strings.Replace(oline, "	", " ", -1)

			for strings.HasPrefix(line, " ") {
					line = strings.TrimPrefix(line, " ")
			}

			if strings.HasPrefix(line, "'!rayonix") {
					//Parse this
					subAppend = process(line, subAppend, filePath)
			} else {
					//Dump it raw
					toAppend = append(toAppend, oline)//+"\r\n")
			}
	}

	//Write subAppend to toAppend
	for lnIndex:=0; lnIndex < len(subAppend); lnIndex++ {
			line := subAppend[lnIndex] //+"\r\n"
			toAppend = append(toAppend, line)
	}

	return toAppend
}

func pMeta(httpPath string, toAppend []string) []string {

	/*
	 * Parses a meta (remote) file (which CANNOT include !rayonix
	 * commands)
	**/

	var data []string      //Input document

	//Get via HTTP
	resp, err := http.Get(httpPath)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	
	//Read the body
	bytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	//Detect newline character
	if strings.Contains(string(bytes), "\r\n") {
		data = strings.Split(string(bytes), "\r\n")
	} else {
		data = strings.Split(string(bytes), "\n")
	}
	
	for lnIndex:=0; lnIndex < len(data); lnIndex++ {
			line := data[lnIndex]
			toAppend = append(toAppend, line)
	}

	return toAppend
}

func process(line string, toAppend []string, filePath string) []string {

	/*
	 * Parses a line's !rayonix command. This is the
	 * central handler.
	 *
	 * Currently supports: "'!rayonix import path/to/file.bas"
	 * and "'!rayonix meta http://www.server.com/file.bas"
	**/

	cmds := strings.Split(line, " ")

	if len(cmds) < 2 {
		return toAppend
	}

	switch(cmds[1]) {
		case "import":

			if len(cmds) < 3 {
				return toAppend
			}

			fp := line[17:len(line)]
			//Standardize filePath
			fp = strings.Replace(fp, "\\", "/", -1)

			//Remove newline characters
			toAppend = pFile(
					strings.Replace(
						strings.Replace(fp, "\r\n" , "", -1),
						"\n","",-1), toAppend, filePath)
			break
			
		case "meta":
			
			if len(cmds) < 3 {
				return toAppend
			}
			
			fp := line[15:len(line)]
				
			//Remove newline characters
			toAppend = pMeta(
				strings.Replace(
					strings.Replace(fp, "\r\n" , "", -1),
					"\n","",-1), toAppend)
			break

		default:
			break
	}

	return toAppend
}

func displayLicense() {

	/*
	 * Display license then quit.
	**/
	
	fmt.Println(`
	Rayonix
	v1.0 (Alpha)
	
    This program is free software; you can redistribute it and/or modify
    it under the terms of the GNU General Public License as published by
    the Free Software Foundation; either version 2 of the License, or
    (at your option) any later version.
   
    This program is distributed in the hope that it will be useful,
    but WITHOUT ANY WARRANTY; without even the implied warranty of
    MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
    GNU General Public License for more details.
   
    You should have received a copy of the GNU General Public License
    along with this program; if not, write to the Free Software
    Foundation, Inc., 51 Franklin Street, Fifth Floor, Boston,
    MA 02110-1301, USA.
    `)
	
}

func displayUsage() {

	/*
	 * Display application usage then quit.
	**/

	fmt.Println(`
	usage: rayonix [option] ...
	Copyright (C) 2019. All Rights Reserved.
	Distributed under the GNU General Public License.
	
	Options:
	rayonix init <projectName> <mainFile.bas>
	   -- initializes a new rayonix project template
	rayonix build <mainFile.bas> <out.bas>
	   -- builds the rayonix project
	rayonix doc
	   -- displays documentation
	rayonix disclaimer
	   -- displays disclaimer
	`)
	os.Exit(1)
}

func displayDocumentation() {

	/*
	 * Display application documentation then quit.
	**/
	
	fmt.Println(`
	Rayonix
	v1.0 (Alpha)
	Copyright (C) 2019. All Rights Reserved.
	Distributed under the GNU General Public License.

	# About

	Rayonix is a JustBasic/LibertyBasic compatible build
	system. It assembles multiple files into one file,
	which makes it great for large projects.
	
	Rayonix IS NOT made by nor endorsed by Shoptalk
	Systems, the copyright owner of LibertyBasic and
	JustBasic.
	
	Rayonix does not make files that no longer work with
	JustBasic or LibertyBasic. Instead, it extends their
	functionality by being a feature-rich preprocessor.
	
	Because of LibertyBasic's cross-platform horizons,
	Rayonix is also cross-platform, thatway anyone who
	uses LibertyBasic can use Rayonix too. Rayonix
	runs Mac, Windows, all Linux, and the Raspberry Pi.
	
	## In-file Flags
	
	Rayonix has these in-file flags, which are included
	as comments:
	  '!rayonix import relative/path/to/file.bas
	  '!rayonix import /absolute/path/to/file.bas
	  '!rayonix meta http://www.website.com/file.bas
	These flags allow you to include code from another
	file, reducing file sizes and enabling the use of
	multiple files.
	
	## Building with Rayonix
	
	To build with Rayonix, you have two options:
	a) Use the GUI made with JustBasic/LibertyBasic
	b) Use a terminal
	
	To use a terminal, simply type:
	  "rayonix build inFile.bas outFile.bas"
	This will build inFile.bas, parsing all infile flags.
	The resulting outFile.bas can be run with JustBasic
	or LibertyBasic.
	
	--Future
	
	In the future, these features may be added:
	-Translation (for example, to Spanish)
	`)
	os.Exit(0)
}
