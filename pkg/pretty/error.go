package pretty

func Error(err error) string {
	return Colorf("[red][bold]Error:[reset] %s", err.Error())
}
