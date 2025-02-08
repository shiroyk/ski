package html

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

func TestParse(t *testing.T) {
	t.Run("fragment", func(t *testing.T) {
		node, err := Parse(`<div id="n1">1</div><div>2</div>`)
		require.NoError(t, err)
		assert.Equal(t, atom.Div, node.FirstChild.DataAtom)
	})
	t.Run("doctype", func(t *testing.T) {
		node, err := Parse(`<!DOCTYPE html><html lang="en"><body><div id="main"></div></body></html>`)
		require.NoError(t, err)
		assert.Equal(t, html.DoctypeNode, node.FirstChild.Type)
	})
	t.Run("document", func(t *testing.T) {
		node, err := Parse(`<html lang="en"><body><div id="main"></div></body></html>`)
		require.NoError(t, err)
		assert.Equal(t, atom.Html, node.FirstChild.DataAtom)
	})
	t.Run("document comment", func(t *testing.T) {
		node, err := Parse(`<!--x--><html lang="en"><body><div id="main"></div></body></html>`)
		require.NoError(t, err)
		assert.Equal(t, atom.Html, node.FirstChild.NextSibling.DataAtom)
	})
	t.Run("doctype comment", func(t *testing.T) {
		node, err := Parse(`<!--x--><!DOCTYPE html><html lang="en"><body><div id="main"></div></body></html>`)
		require.NoError(t, err)
		assert.Equal(t, html.DoctypeNode, node.FirstChild.NextSibling.Type)
	})
	t.Run("fragment comment", func(t *testing.T) {
		node, err := Parse(`<!--x--><div id="n1">1</div><div>2</div>`)
		require.NoError(t, err)
		assert.Equal(t, atom.Div, node.FirstChild.NextSibling.DataAtom)
	})
}
