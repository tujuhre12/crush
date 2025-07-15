package ollama

import (
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"
)

var processManager = &ProcessManager{
	processes: make(map[string]*exec.Cmd),
}

// setupProcessCleanup sets up signal handlers to clean up processes on exit
func setupProcessCleanup() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-c
		cleanupProcesses()
		os.Exit(0)
	}()
}

// cleanupProcesses terminates all Ollama processes started by Crush
func cleanupProcesses() {
	processManager.mu.Lock()
	defer processManager.mu.Unlock()

	// Clean up model processes
	for modelName, cmd := range processManager.processes {
		if cmd.Process != nil {
			cmd.Process.Kill()
			cmd.Wait() // Wait for the process to actually exit
		}
		delete(processManager.processes, modelName)
	}

	// Clean up Ollama server if Crush started it
	if processManager.crushStartedOllama && processManager.ollamaServer != nil {
		if processManager.ollamaServer.Process != nil {
			// Kill the entire process group to ensure all children are terminated
			syscall.Kill(-processManager.ollamaServer.Process.Pid, syscall.SIGTERM)

			// Give it a moment to shut down gracefully
			time.Sleep(2 * time.Second)

			// Force kill if still running
			if processManager.ollamaServer.ProcessState == nil {
				syscall.Kill(-processManager.ollamaServer.Process.Pid, syscall.SIGKILL)
			}

			processManager.ollamaServer.Wait() // Wait for the process to actually exit
		}
		processManager.ollamaServer = nil
		processManager.crushStartedOllama = false
	}
}
