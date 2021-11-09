package data

import "testing"

func TestCheckValidation(t *testing.T) {

	person := &Person{
		Firstname: "David",
		Lastname:  "Hernandez",
	}

	err := person.Validate()

	if err != nil {
		t.Fatal(err)
	}

}
