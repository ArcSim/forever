package hunter

// The 15% quiver the site lists is Harpy Hide Quiver, which a player has before raiding. The generator keeps the highest
// item level at each speed, Ancient Sinew Wrapped Lamina (the epic quest's); both grant the same 15% (spell 29414).
func init() {
	lamina := quivers["AncientSinewWrappedLamina"]
	delete(quivers, "AncientSinewWrappedLamina")
	quivers["HarpyHideQuiver"] = generatedQuiver{Name: "Harpy Hide Quiver", ItemID: 19319, Haste: lamina.Haste}
}
