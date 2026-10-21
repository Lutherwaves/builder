## <RULE-ID>: <short name>

<!--
One block per rule. Keep the four lines in this order.
The guard's failure message quotes <RULE-ID> and this file, so the person who trips it lands here.
-->

**Rule.** <One sentence, no more. A reviewer can point at a diff and say "this breaks it" or "this keeps it".>

**Why.** <The incident, its scar: date, what changed, what broke, for whom, how long. Link the write-up or the commit.>
<If it happened more than once, list each time. A repeated correction counts: name the date and what was corrected.>

**Guard.** <The test or check that fails when someone breaks this rule, and the command that runs it.>
<If none exists: "none yet — add one", plus the check you propose.>
<Name what the guard does not catch.>

**Supersedes.** <The RULE-ID this rule replaces, with a link. Or "none".>
