package util

import (
	"fmt"
	"math/rand"
)

var adjectives = []string{
	"Fast", "Slow", "Quick", "Speedy", "Trotting", "Weaving", "Waiving", "Gracious", "Healthy", "Happy", "Funny",
	"Red", "Blue", "Green", "Orange", "Purple", "Fuzzy", "Smiling", "Tall", "Grand", "Ultimate", "Prime",
	"Alpha", "Growling", "Slithering", "Swimming", "Flying", "Jumping", "Running", "Charging", "Shooting", "Bouncing",
	"Bounding", "Leaping",
}

var animals = []string{
	"Dog", "Cat", "Mouse", "Alligator", "Crocodile", "Shark", "Hippo", "Giraffe", "Antelope", "Lion", "Tiger",
	"Bear", "Muskrat", "Otter", "Dolphin", "Porcupine", "Gerbil", "Hedgehog", "Snake", "Lizard", "Chipmunk",
	"Bird", "Dinosaur", "Okapi", "Eagle", "Mandrill", "Bonobo", "Wolf", "Fox", "Armadillo", "Rhino", "Anteater",
	"Reindeer", "Deer", "Panda",
}

// GetRandomName returns a random name by combining an adjective with an animal
func GetRandomName() string {
	adjectivesIndex := rand.Intn(len(adjectives))
	animalsIndex := rand.Intn(len(animals))

	return fmt.Sprintf("%s %s", adjectives[adjectivesIndex], animals[animalsIndex])
}
