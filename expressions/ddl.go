package expressions

func Create(args Args) Expression  { return newNode(KindCreate, args) }
func Command(args Args) Expression { return newNode(KindCommand, args) }
