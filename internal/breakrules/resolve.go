package breakrules

import "fmt"

// Resolve expands all $variable references in a RuleSet by substituting
// the variable's expression tree. It detects cycles and undefined variables.
func Resolve(rs *RuleSet) []error {
	defs := make(map[string]*Node, len(rs.Assignments))
	for _, a := range rs.Assignments {
		defs[a.Name] = a.Expr
	}

	var errs []error
	resolved := make(map[string]bool)
	resolving := make(map[string]bool)

	var resolveNode func(n *Node) *Node

	var resolveVar func(name string) (*Node, error)
	resolveVar = func(name string) (*Node, error) {
		if resolved[name] {
			return defs[name], nil
		}
		if resolving[name] {
			return nil, fmt.Errorf("cycle detected in variable $%s", name)
		}
		expr, ok := defs[name]
		if !ok {
			return nil, fmt.Errorf("undefined variable $%s", name)
		}
		resolving[name] = true
		result := resolveNode(expr)
		delete(resolving, name)
		defs[name] = result
		resolved[name] = true
		return result, nil
	}

	resolveNode = func(n *Node) *Node {
		if n == nil {
			return nil
		}
		switch n.Kind {
		case NodeVariable:
			expr, err := resolveVar(n.Name)
			if err != nil {
				errs = append(errs, err)
				return n
			}
			return cloneNode(expr)
		case NodeConcat, NodeAlt:
			out := &Node{Kind: n.Kind, Children: make([]*Node, len(n.Children))}
			for i, c := range n.Children {
				out.Children[i] = resolveNode(c)
			}
			return out
		case NodeStar, NodePlus, NodeQuest:
			return &Node{Kind: n.Kind, Child: resolveNode(n.Child)}
		default:
			return n
		}
	}

	for _, a := range rs.Assignments {
		if !resolved[a.Name] {
			expr, err := resolveVar(a.Name)
			if err != nil {
				errs = append(errs, err)
			} else {
				a.Expr = expr
			}
		}
	}

	for _, r := range rs.Rules {
		r.Expr = resolveNode(r.Expr)
	}

	if len(errs) > 0 {
		return errs
	}
	return nil
}

func cloneNode(n *Node) *Node {
	if n == nil {
		return nil
	}
	c := &Node{
		Kind:    n.Kind,
		Classes: n.Classes,
		Name:    n.Name,
		Tag:     n.Tag,
	}
	if n.Child != nil {
		c.Child = cloneNode(n.Child)
	}
	if len(n.Children) > 0 {
		c.Children = make([]*Node, len(n.Children))
		for i, ch := range n.Children {
			c.Children[i] = cloneNode(ch)
		}
	}
	return c
}
