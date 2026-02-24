package main

var botNames = []string{
	"Alice", "Bob", "Charlie", "Diana", "Eve", "Frank",
	"Grace", "Hank", "Ivy", "Jack", "Karen", "Leo",
	"Mona", "Nick", "Olivia", "Pete", "Quinn", "Rosa",
	"Sam", "Tina", "Uma", "Vic", "Wendy", "Xander",
}

func randomName(index int) string {
	if index < len(botNames) {
		return botNames[index]
	}
	return botNames[cryptoIntn(len(botNames))]
}
