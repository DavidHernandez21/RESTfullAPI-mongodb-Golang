package data

import "testing"

func TestCheckValidationPerson(t *testing.T) {

	person := &Person{
		Firstname: "David",
		Lastname:  "Hernandez",
	}

	err := person.Validate()

	if err != nil {
		t.Fatal(err)
	}

}

func TestCheckValidationPersonUpdate(t *testing.T) {

	person := &PersonUpdate{
		Firstname: "David",
		Lastname:  "",
	}

	err := person.Validate()

	if err != nil {
		t.Fatal(err)
	}

}
