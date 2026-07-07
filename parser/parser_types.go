package parser

import (
	exp "github.com/sjincho/sqlglot-go/expressions"
	"github.com/sjincho/sqlglot-go/tokens"
)

func (p *Parser) parseTypes() exp.Expression {
	index := p.index
	if !p.matchSet(typeTokens) {
		return nil
	}
	typeToken := p.prev.TokenType
	var expressions []exp.Expression
	if p.match(tokens.L_PAREN) {
		expressions = p.parseCsv(p.parseTypeSize)
		if !p.match(tokens.R_PAREN) {
			p.retreat(index)
			return nil
		}
	}
	dtype, ok := exp.DTypeFromName(tokens.TypeName(typeToken))
	if !ok {
		p.retreat(index)
		return nil
	}
	args := exp.Args{"this": dtype}
	if len(expressions) > 0 {
		args["expressions"] = expressions
	}
	return p.expression(exp.DataType(args), nil, nil)
}

func (p *Parser) parseTypeSize() exp.Expression {
	this := p.parseType()
	if this == nil {
		return nil
	}
	if this.Kind() == exp.KindColumn && this.Arg("table") == nil {
		this = exp.Var(exp.Args{"this": stringsUpper(this.Name())})
	}
	return p.expression(exp.DataTypeParam(exp.Args{"this": this, "expression": p.parseVar(true, nil, false)}), nil, nil)
}

func (p *Parser) parseCast(strict bool, safe any) exp.Expression {
	this := p.parseAssignment()
	if !p.match(tokens.ALIAS) {
		if p.match(tokens.COMMA) {
			return p.expression(exp.CastToStrType(exp.Args{"this": this, "to": p.parseString()}), nil, nil)
		}
		p.raiseError("Expected AS after CAST")
	}
	to := p.parseTypes()
	var default_ exp.Expression
	if p.match(tokens.DEFAULT) {
		default_ = p.parseBitwise()
		p.matchTextSeq("ON", "CONVERSION", "ERROR")
	}
	if to == nil {
		p.raiseError("Expected TYPE after CAST")
	}
	args := exp.Args{"this": this, "to": to, "default": default_, "action": p.parseVarFromOptions(castActions, false)}
	if safe != nil {
		args["safe"] = safe
	}
	return p.buildCast(strict, args)
}

func (p *Parser) buildCast(strict bool, args exp.Args) exp.Expression {
	kind := exp.KindCast
	if !strict {
		kind = exp.KindTryCast
		if p.dialect.TryCastRequiresString != nil {
			args["requires_string"] = *p.dialect.TryCastRequiresString
		}
	}
	return p.Expression(exp.New(kind, args))
}

func (p *Parser) parseDcolon() exp.Expression {
	return p.parseTypes()
}

func (p *Parser) parseString() exp.Expression {
	if p.match(tokens.STRING) {
		return p.expression(exp.LiteralString(p.prev.Text), &p.prev, nil)
	}
	return p.parsePlaceholder()
}

func (p *Parser) parseVar(anyToken bool, toks map[tokens.TokenType]bool, upper bool) exp.Expression {
	if (anyToken && p.advanceAny(false) != nil) || p.match(tokens.VAR) || (toks != nil && p.matchSet(toks)) {
		text := p.prev.Text
		if upper {
			text = stringsUpper(text)
		}
		return p.expression(exp.Var(exp.Args{"this": text}), nil, nil)
	}
	return p.parsePlaceholder()
}

func (p *Parser) parseVarOrString(upper bool) exp.Expression {
	// Mirror upstream `self._parse_string() or self._parse_var(...)`: short-circuit
	// so parseVar's advanceAny doesn't eagerly consume the token after a matched
	// string literal (e.g. the FROM in EXTRACT('lit' FROM x)).
	if s := p.parseString(); s != nil {
		return s
	}
	return p.parseVar(true, nil, upper)
}

func (p *Parser) parseBracket(this exp.Expression) exp.Expression {
	if !p.match(tokens.L_BRACKET) {
		return this
	}
	expressions := p.parseCsv(p.parseDisjunction)
	if !p.match(tokens.R_BRACKET) {
		p.raiseError("Expected ]")
	}
	if this == nil {
		this = exp.Array(exp.Args{"expressions": expressions})
	} else {
		this = p.expression(exp.Bracket(exp.Args{"this": this, "expressions": expressions}), nil, this.PopComments())
	}
	p.addComments(this)
	return p.parseBracket(this)
}
