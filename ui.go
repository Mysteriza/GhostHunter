package main

import "github.com/fatih/color"

// displayWelcomeMessage shows the welcome banner
func displayWelcomeMessage() {
	color.Cyan(`
  ____ _               _   _   _             _            
 / ___| |__   ___  ___| |_| | | |_   _ _ __ | |_ ___ _ __ 
| |  _| '_ \ / _ \/ __| __| |_| | | | | '_ \| __/ _ \ '__|
| |_| | | | | (_) \__ \ |_|  _  | |_| | | | | ||  __/ |   
 \____|_| |_|\___/|___/\__|_| |_|\__,_|_| |_|\__\___|_|      
	`)
	color.Green("\nWelcome to GhostHunter!")
	color.Yellow("Unearth hidden treasures from the Wayback Machine!")
	color.Yellow("Let's hunt some ghosts! 👻")
	color.Cyan("\nCreated by: Mysteriza & Deepseek AI")
	color.Cyan("-------------------------------------------------------------")
}