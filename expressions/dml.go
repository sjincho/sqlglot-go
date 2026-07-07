package expressions

func Insert(args Args) Expression     { return newNode(KindInsert, args) }
func Update(args Args) Expression     { return newNode(KindUpdate, args) }
func Delete(args Args) Expression     { return newNode(KindDelete, args) }
func Merge(args Args) Expression      { return newNode(KindMerge, args) }
func When(args Args) Expression       { return newNode(KindWhen, args) }
func Whens(args Args) Expression      { return newNode(KindWhens, args) }
func OnConflict(args Args) Expression { return newNode(KindOnConflict, args) }
func Returning(args Args) Expression  { return newNode(KindReturning, args) }
