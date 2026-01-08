package main

import (
    "embed"
    "fmt"
    "io/fs"
    "os"
    "path/filepath"
    "strings"
    "regexp"
    "os/exec"
    "runtime"

    "github.com/AlecAivazis/survey/v2"
)

//go:embed templates/*
var templateFS embed.FS

type Config struct {
    AppName     string
    PackageName string
    MinSdk      string
}

func validatePackageName(val interface{}) error {
    str, ok := val.(string)
    if !ok {
        return fmt.Errorf("package name must be a string")
    }
    
    // Basic package name validation
    matched, _ := regexp.MatchString(`^[a-z][a-z0-9_]*(\.[a-z][a-z0-9_]*)*$`, str)
    if !matched {
        return fmt.Errorf("invalid package name format. Use: com.example.appname")
    }
    return nil
}

// FindJava locates the java executable
func FindJava() string {
    path, err := exec.LookPath("java")
    if err != nil {
        return "Not Found"
    }
    return path
}

// FindSDK checks standard Android environment variables
func FindSDK() string {
    // Check both standard variables
    path := os.Getenv("ANDROID_HOME")
    if path == "" {
        path = os.Getenv("ANDROID_SDK_ROOT")
    }
    if path == "" {
        return "Not Found (Check ANDROID_HOME)"
    }
    return path
}

// FindGradle locates global gradle (if installed)
func FindGradle() string {
    path, err := exec.LookPath("gradle")
    if err != nil {
        return "Not Found (Will use Wrapper)"
    }
    return path
}

const (
    AGPVersion        = "8.13.2"
    KotlinVersion     = "2.3.0"
    GradleVersion     = "9.2.1"
    ComposeBomVersion = "2025.12.01"
    MinSdk            = "24"
)

func main() {
    // --- Environment Check ---
    fmt.Println("🔍 Checking Environment...")

    javaPath := FindJava()
    sdkPath := FindSDK()
    gradlePath := FindGradle()

    // Print status
    if javaPath == "Not Found" {
        fmt.Println("⚠️  Java:", javaPath)
    } else {
        fmt.Println("✅ Java:", javaPath)
    }

    if strings.Contains(sdkPath, "Not Found") {
        fmt.Println("⚠️  Android SDK:", sdkPath)
    } else {
        fmt.Println("✅ Android SDK:", sdkPath)
    }

    if strings.Contains(gradlePath, "Not Found") {
        fmt.Println("ℹ️  Gradle:", gradlePath)
    } else {
        fmt.Println("✅  Gradle:", gradlePath)
    }
    
    fmt.Println("")
    fmt.Println("🔨 Build System Versions")
    fmt.Println("ℹ️  Target AGP Version:", AGPVersion)
    fmt.Println("ℹ️  Target Kotlin Version:", KotlinVersion)
    fmt.Println("ℹ️  Target Gradle Wrapper:", GradleVersion)

    fmt.Println(strings.Repeat("-", 50))
    // -------------------------------

	var answers Config
	
	qs := []*survey.Question{
        {
            Name:     "AppName",
            Prompt:   &survey.Input{Message: "What is the App Name?", Default: "Mad"},
            Validate: survey.Required,
        },
        {
            Name:     "PackageName",
            Prompt:   &survey.Input{Message: "Package Name?", Default: "com.example.myapp"},
            Validate: survey.ComposeValidators(survey.Required, validatePackageName),
        },
        {
            Name: "MinSdk",
            Prompt: &survey.Select{
                Message: "Select minimum SDK:",
                Options: []string{"21", "22", "23", "24", "25", "26", "27", 
                                  "28", "29", "30", "31", "32", "33", "34",
                                  "35", "36"},
                Default: MinSdk,
            },
        },
    }

	err := survey.Ask(qs, &answers)
    if err != nil {
        fmt.Println("❌ Error:", err)
        return
    }

	outputDir := answers.AppName

    if _, err := os.Stat(outputDir); err == nil {
        overwrite := false
        prompt := &survey.Confirm{
            Message: fmt.Sprintf("Directory '%s' already exists. Overwrite?", outputDir),
        }
        survey.AskOne(prompt, &overwrite)
        if !overwrite {
            fmt.Println("❌ Operation cancelled.")
            return
        }
        // Remove existing directory
        os.RemoveAll(outputDir)
    }

	targetPackagePath := strings.ReplaceAll(answers.PackageName, ".", "/")
    
	fmt.Printf("🚀 Generating %s in ./%s ...\n", answers.AppName, outputDir)

    // Make the root dir
    if err := os.MkdirAll(outputDir, 0755); err != nil {
        fmt.Printf("\n❌ Failed to create project directory: %v\n", err)
        return
    }

	err = fs.WalkDir(templateFS, "templates", func(path string, d fs.DirEntry, err error) error {	
        if err != nil {
            return err
        }

        if path == "templates" {
            return nil
        }

        // Remove the "templates/" prefix to get the relative path
        relPath := strings.TrimPrefix(path, "templates/")

        // Define our hardcoded template paths
        const sourcePackagePath = "com/example/rockstarcompose"
        const javaSourceBase = "app/src/main/java/"

        // Default destination path (unless we modify it below)
        destPath := filepath.Join(outputDir, relPath)

        if strings.HasPrefix(relPath, javaSourceBase) {
            // CASE 1: We are inside the package folder (or deeper)
            // If the path contains "com/example/app", we perform the swap
            if strings.Contains(relPath, sourcePackagePath) {
                // Find exactly where "com/example/app" starts
                idx := strings.Index(relPath, sourcePackagePath)
                if idx != -1 {
                    // Reconstruct path: Prefix + NewPackage + Suffix
                    prefix := relPath[:idx]
                    suffix := relPath[idx+len(sourcePackagePath):]
                    
                    newRelPath := prefix + targetPackagePath + suffix
                    destPath = filepath.Join(outputDir, newRelPath)
                }
            // we are likely visiting the parent directories: "com" or "com/example"
            } else if answers.PackageName != "com.example.app" {
                // If the user's package is DIFFERENT from the template, we must skip 
                // creating the old template parents (com/example) to avoid empty junk folders.
                if relPath == "app/src/main/java/com" || relPath == "app/src/main/java/com/example" {
                    // Return nil to skip this directory entry
                    return nil
                }
            }
        }
        
        if d.IsDir() {
            // Create directory
           //  fmt.Println("destPath", destPath)
            return os.MkdirAll(destPath, 0755)
        }

        // Read file content
        content, err := templateFS.ReadFile(path)
        if err != nil {
            return err
        }

        // Handle Permissions
        // Default to read/write for owner/group
        perm := fs.FileMode(0644) 

        if filepath.Base(destPath) == "gradlew" {
            perm = 0755 
        }

        // Replace placeholders in the file content
        fileStr := string(content)
        fileStr = strings.ReplaceAll(fileStr, "{{APP_NAME}}", answers.AppName)
        fileStr = strings.ReplaceAll(fileStr, "{{PACKAGE_NAME}}", answers.PackageName)
        fileStr = strings.ReplaceAll(fileStr, "{{MIN_SDK}}", answers.MinSdk)

        fileStr = strings.ReplaceAll(fileStr, "{{GRADLE_VERSION}}", GradleVersion)
        fileStr = strings.ReplaceAll(fileStr, "{{AGP_VERSION}}", AGPVersion)
        fileStr = strings.ReplaceAll(fileStr, "{{KOTLIN_VERSION}}", KotlinVersion)
        fileStr = strings.ReplaceAll(fileStr, "{{CBOM_VERSION}}", ComposeBomVersion)

        // Write to destination
        return os.WriteFile(destPath, []byte(fileStr), perm)
    })

    if err != nil {
        fmt.Printf("\n❌ Failed to generate project: %v\n", err)
        return
    }

    // 4. Create a helpful README (Optional but nice)
    readmeContent := fmt.Sprintf(`# %s

Generated by droid-cli.

## To build this project

1. Make sure you have the Android SDK installed.
2. Then, set ANDROID_HOME to the path you installed SDK on.
3. Run:
   ./gradlew build
4. adb shell am start -n your.package.id/.MainActivity
   or
   adb shell monkey -p your.package.id -c android.intent.category.LAUNCHER 1
`, answers.AppName)
    
    os.WriteFile(filepath.Join(outputDir, "README.md"), []byte(readmeContent), 0644)
    fmt.Println("\n✅ Success!")

    absPath, err := filepath.Abs(outputDir)
    if err != nil {
        absPath = outputDir
    }
    fmt.Printf("📂 $ cd %s\n", absPath)

    switch runtime.GOOS {
	case "windows":
		fmt.Println("🔨 $ gradle buildDebug")
	case "linux", "darwin":
		fmt.Println("🔨 $ ./gradlew buildDebug")
	default:
		fmt.Println("🔨 $ ./gradlew buildDebug")
	}
}