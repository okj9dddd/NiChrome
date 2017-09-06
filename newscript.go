package main

//include a loading bar
//TODO: Use filePath.join
//automated "yes"er updating the config file
//TODO check if it is a device
//TODO append method name to error
//TODO proper output channels when you run commands
//TODO in the newest kernel pull the stable one if it fails, then go back to what was there, see the notes on the PR)
import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
)

var (
	fetch         = flag.Bool("fetch", true, "Fetch all the things we need")
	keys          = flag.String("keys", "vboot_reference/tests/devkeys", "where the keys live")
	kernelVersion = "4.12.7"
	workingDir    = ""
	linuxVersion  = "linux_stable"
	homeDir       = ""
	totalSteps    = 7
	packageList   = []string{
		"git", "golang", "build-essential", "git-core", "gitk", "git-gui", "subversion", "curl", "python2.7", "libyaml-dev", "liblzma-dev"}
)

func cp(inputLoc string, outputLoc string) error {
	if _, err := os.Stat(inputLoc); err != nil {
		return err
	}
	fileContent, err := ioutil.ReadFile(inputLoc)
	if err != nil {
		return err
	}
	ioutil.WriteFile(outputLoc, fileContent, 0777)
	return nil
}

func tildeExpand(input string) string {
	if strings.Contains(input, "~") {
		input = strings.Replace(input, "~", homeDir, 1)
		fmt.Printf("Full filepath is : %s", input)
	}
	return input
}

func setup() error {
	fmt.Printf(fmt.Sprintf("-------- Setting up (1/%d)\n", totalSteps))
	dir, err := os.Getwd()
	if err != nil {
		return err
	}
	workingDir = dir
	fmt.Printf("Working dir is %s\n", workingDir)
	usr, err := user.Current()
	if err != nil {
		return err
	}
	homeDir = usr.HomeDir
	fmt.Printf("Home dir is %s\n", homeDir)
	fmt.Printf("Using apt-get to get %v\n", packageList)
	get := []string{"apt-get", "install"}
	get = append(get, packageList...)
	cmd := exec.Command("sudo", get...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		return err
	}
	/*err = blankBootstick()
	if err != nil {
		return err
	}*/
	return nil

}

//User input for putting custom chrome image on bootstick
func blankBootstick() error {
	fmt.Printf("-------- Creating bootstick \n")
	var imageLoc = ""
	var location = "/dev/sda"
	for true {
		fmt.Printf("What image would you like to put onto your bootstick (provide location for iso file)?\n")
		_, err := fmt.Scanf("%s", &imageLoc)
		if err != nil {
			return err
		}
		imageLoc = tildeExpand(imageLoc)
		if err != nil {
			return err
		}
		if _, err = os.Stat(imageLoc); err != nil {
			fmt.Printf("Please provide a valid file name. %s has error %v\n", imageLoc, err)
		} else {
			break
		}
	}
	for true {
		fmt.Printf("Where is your bootstick (%s)?\n", location)
		_, err := fmt.Scanf("%s", &location)
		if err != nil {
			return err
		}
		location = tildeExpand(location)
		if err != nil {
			return err
		}
		if _, err = os.Stat(location); err != nil {
			fmt.Printf("Please provide a valid location name. %s has error %v\n", location, err)
		} else {
			break
		}
	}
	fmt.Printf("Running dd to put the new image onto the desired location. \n")
	if err := cp(imageLoc, location); err != nil {
		return err
	}
	return nil
}

func cleanup() error {
	filesToRemove := [...]string{linuxVersion, "linux-stable", "NiChrome", "vboot_reference"}
	fmt.Printf("-------- Removing problematic files %v\n", filesToRemove)
	for _, file := range filesToRemove {
		if _, err := os.Stat(file); err != nil {
			if os.IsNotExist(err) {
				continue
			}
		}
		err := os.RemoveAll(file)
		if err != nil {
			return err
		}
	}
	return nil
}

func goCompatibility() error {
	fmt.Printf("--------Checking Go Compatibility \n")
	cmd := exec.Command("go", "version")
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return err
	}
	// The string is originally in the form: go version go1.9rc2_cl165246139 linux/amd64 where 1.9 is the go version
	termString, err := strconv.ParseFloat(strings.Split(out.String(), " ")[2][2:5], 64)
	if err != nil {
		return err
	}
	if termString > 1.7 {
		fmt.Println("Compatible go version")
	} else {
		return errors.New("Please install go v1.7 or greater.")
	}
	return nil
}

func goGet() error {
	fmt.Printf("--------Getting u-root \n")
	cmd := exec.Command("sh", "-c", "export", "GOPATH=\"$HOME/go\"")
	err := cmd.Run()
	if err != nil {
		return err
	}
	fmt.Printf("exported \n")
	cmd = exec.Command("go", "get", "github.com/u-root/u-root/")
	err = cmd.Run()
	/*if err != nil {
		return err
	}*/
	fmt.Printf("--------Got u-root \n")
	gopath := fmt.Sprintf("GOPATH=%s/go", homeDir)
	bbpath := fmt.Sprintf("%s/go/src/github.com/u-root/u-root/bb/bb", homeDir)
	cmd = exec.Command("go", "build", "bbpath")
	err = cmd.Run()
	if err != nil {
		return err
	}
	cmd = exec.Command(gopath, bbpath)
	err = cmd.Run()
	if err != nil {
		return err
	}
	fmt.Printf("--------Getting bb \n")
	if _, err := os.Stat("/tmp/initramgs.linux_amd64.cpio"); err != nil {
		return err
	}
	fmt.Printf("Created the initramfs in /tmp/")
	return nil
}

func kernelGet() error {
	var args = []string{"clone", "--depth", "1", "-b", "v4.12.7", "git://git.kernel.org/pub/scm/linux/kernel/git/stable/linux-stable.git"}
	fmt.Printf("-------- Getting the kernel via git %v\n", args)
	cmd := exec.Command("git", args...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Printf("didn't clone kernel %v", err)
		return err
	}
	return nil
}

func unpackKernel() error {
	if err := os.Symlink("/tmp/initramfs.linux_amd64.cpio", fmt.Sprintf("%s/initramfs.linux_amd64.cpio", "linux-stable")); err != nil {
		fmt.Printf("[warning only] Error creating symlink for initramfs: %v", err)
	}
	// NOTE: don't get confused. This means that .config in linux-stable
	// points to CONFIG, i.e. where we are.
	if err := os.Symlink("../CONFIG", "linux-stable/.config"); err != nil {
		fmt.Printf("[warning only] Error creating symlink for .config: %v", err)
	}

	cmd := exec.Command("make", "--directory", "linux-stable", "-j64")
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	err := cmd.Run()
	if err != nil {
		return err
	}
	if _, err := os.Stat(filepath.Join("linux-stable", "/arch/x86/boot/bzImage")); err != nil {
		return err
	}
	fmt.Printf("bzImage created")
	return nil
}

func buildVbutil() error {
	fmt.Printf("-------- Building in Vbutil\n")
	if *fetch {
		cmd := exec.Command("git", "clone", "https://chromium.googlesource.com/chromiumos/platform/vboot_reference")
		cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
		err := cmd.Run()
		if err != nil {
			fmt.Printf("didn't get chromium repo")
			return err
		}
	}
	cmd := exec.Command("git", "checkout", "3f3a496a23088731e4ab5654b02fbc13a6881c65")
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	cmd.Dir = "vboot_reference"
	if err := cmd.Run(); err != nil {
		fmt.Printf("couldn't checkout the right branch")
		return err
	}
	cmd = exec.Command("make", "-j64")
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	cmd.Dir = "vboot_reference"
	if err := cmd.Run(); err != nil {
		fmt.Printf("Make failed. Please try to manually install vbutil")
		return err
	}
	return nil

}

func vbutilIt() error {
	fmt.Printf("-------- VBUTILING\n")
	buildVbutil()
	fmt.Printf("-------- VBUTILING  contd. \n")
	newKern := "newKern"
	if err := ioutil.WriteFile("config.txt", []byte("loglevel=7"), 0777); err != nil {
		return err
	}
	if err := ioutil.WriteFile("nocontent.efi", []byte("no content"), 0777); err != nil {
		return err
	}
	bzImage := "linux-stable/arch/x86/boot/bzImage"
	fmt.Printf("Bz image is located at %s \n", bzImage)
	keyblock := filepath.Join(*keys, "kernel.keyblock")
	sign := filepath.Join(*keys, "kernel_data_key.vbprivk")
	cmd := exec.Command("./vboot_reference/build/futility/futility", "vbutil_kernel", "--pack", newKern, "--keyblock", keyblock, "--signprivate", sign, "--version", "1", "--vmlinuz", bzImage, "--bootloader", "nocontent.efi", "--config", "config.txt", "--arch", "x86")
	stdoutStderr, err := cmd.CombinedOutput()
	fmt.Printf("%s\n", stdoutStderr)
	if err != nil {
		return err
	}
	if err = dd(); err != nil {
		return err
	}
	return nil
}

func dd() error {
	var location = "/dev/sda2"
	for true {
		fmt.Printf("Where do you want to put this kernel (%s)", location)
		_, err := fmt.Scanf("%s", &location)
		if err != nil {
			return err
		}
		if _, err = os.Stat(location); err != nil {
			fmt.Printf("Please provide a valid location name. %s has error %v", location, err)
		} else {
			break
		}
	}
	fmt.Printf("Running dd to put the new kernel onto the desired location on the usb.\n")
	args := []string{"dd", "if=newKern", "of=" + location}
	msg, err := exec.Command("sudo", args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("dd %v failed: %v: %v", args, string(msg), err)
	}
	fmt.Printf("%v ran ok\n", args)
	return nil
}

//TODO : final Error
//TODO: absolute filepath things
func allFunc() error {

	if *fetch {
		if err := cleanup(); err != nil {
			log.Printf("ERROR: %v\n", err)
		}
	}
	if err := setup(); err != nil {
		log.Printf("ERROR: %v\n", err)
	}
	if false {
		if err := goCompatibility(); err != nil {
			log.Printf("ERROR: %v\n", err)
		}
	}
	//error ridden
	/*if err := goGet(); err != nil {
		log.Printf("ERROR: %v\n", err)
	}*/
	if *fetch {
		if err := kernelGet(); err != nil {
			log.Printf("ERROR: %v\n", err)
		}
	}
	if err := unpackKernel(); err != nil {
		log.Printf("ERROR: %v\n", err)
	}
	if err := vbutilIt(); err != nil {
		log.Printf("ERROR: %v\n", err)
	}
	return nil
}

func main() {
	flag.Parse()
	//all paramters: name of new kernel, location for dd, kernel version,
	fmt.Printf("Using kernel default as 4.12.7\n")
	if err := allFunc(); err != nil {
		fmt.Printf("fail error is : %v", err)
		os.Exit(1)
	}
	fmt.Printf("execution completed successfully\n")

}
