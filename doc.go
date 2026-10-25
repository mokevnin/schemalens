// Package schemalens checks a live database against rules, and optionally
// against a model manifest, and reports every disagreement as a structured
// Finding. It is the library behind the schemalens command line: the CLI is a
// thin layer over Check, so both produce the same findings, the same JSON and
// the same SARIF.
//
// The interface is small. Check inspects the database named by Options.DSN
// (sqlite://, postgres:// or mysql://), runs the enabled rules and returns a
// Report. The rules themselves and the database inspection are internal.
//
//	rep, err := schemalens.Check(ctx, schemalens.Options{DSN: "sqlite://app.db"})
//	if err != nil {
//		return err // unusable DSN, manifest or configuration
//	}
//	for _, f := range rep.Findings {
//		fmt.Println(f.Rule, f.Message)
//	}
//	if !rep.Passed() {
//		os.Exit(1)
//	}
//
// A model comes from a manifest, a versioned JSON document that an adapter
// prints for an ORM (see ParseManifest and ManifestSchema); with one, the
// model-versus-database rules run too. Behaviour is tuned with a Config, which
// has the same schema as schemalens.yml: rules can be switched off, rules take
// options, and findings are silenced by exclusions that each carry a reason.
// An exclusion that matches no finding is reported in
// Report.UnmatchedExclusions and fails the report, so stale exclusions do not
// accumulate.
//
// Rules and Explain describe the rule set, and Baseline records the current
// findings as exclusions. Report.WriteJSON, Report.WriteText and Report.WriteSARIF
// render a report exactly as the CLI prints it.
//
// The root module depends on no ORM; adapters live in separate modules.
package schemalens
