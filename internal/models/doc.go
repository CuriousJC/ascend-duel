// Package models is plain data structs with no behavior. Constructors only.
//
// It is half of the split this game builds widgets with: a struct here, its behavior beside it in
// internal/systems as Update* and Draw* free functions taking (gs, ...). Button with
// systems.UpdateButton and systems.DrawButton is the reference example, and a new widget follows
// it — a plain struct here, its behavior there, owned by the scene that uses it.
//
// There is no UI toolkit dependency, and that is a decision rather than a gap. ebitenui was
// evaluated and declined: everything this game needs is a *game* widget, which is where
// general-purpose toolkits are weakest, and a toolkit is one more dependency to carry. The
// single trigger for revisiting it is the seed text field
// — a text input with a caret, selection and clipboard is the one widget genuinely cheaper to take
// than to build.
package models
