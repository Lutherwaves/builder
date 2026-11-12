# The architecture lens of a plan review

`plan` loads this file in step 4, after the engineering lens, when the work touches one of four things:
an API that another program calls, a boundary between services or modules, a data store, or a tool that an agent can call.
`review` checks a change against the same questions.

The project's own rules win. Read its guides and its neighbouring code first. Where the project answers a question, follow the project and write "the project's rule" as the finding.
This lens speaks only where the project is silent. It never tells you to rewrite a pattern that is already there.
Write each finding under "Lens findings" in the plan. Each finding changes the plan, or says "no change, because <reason>".

## The contract

1. **Where is the contract written?** Keep the API description next to the code it describes. Generate the spec and the clients from it, and commit them.
   A build step regenerates them and fails on any difference. A contract that a person keeps in step by hand is already wrong.
2. **Is each operation about one resource?** A service offers reads, writes and state changes on one resource.
   A batch or a composed call belongs to the client or to the tool that wraps the API: the shape of an API call is not the shape of a tool.
3. **Does each list have the one list shape?** One filter expression, a limit, an opaque cursor, and one envelope.
   A caller that needs all rows follows the cursor to the end. Never plan a large fixed limit: it cuts the data in silence, and every total over it is wrong.
4. **How does a caller ask for more?** Related or computed data is opt-in, through one parameter with a closed list of keys for each endpoint. An unknown key is an error.
   One operation has one response type. A parameter can add optional fields. It never changes the type.
5. **Is the write input a different shape from the stored record?** A resource has a stored record, a create input and an update input.
   The server sets the id, the owner and the timestamps, also after a partial update. A new field goes into every shape, the validation and the mapping in one change.
6. **Is there one error shape?** One place turns an error into a status and a body. The plan gives the status of each failure.
   A delete that something still needs is refused, and the answer says what blocks it. It never removes the dependants in silence.
7. **Who reads the contract text?** People and agents read each description. Describe each operation, each field and each allowed value in plain words, with an example body.

## The boundaries

8. **Who owns each fact?** One module or service owns each kind of fact. The others ask the owner through its public contract.
   They never read its tables, and they never keep a copy.
9. **Which way do the dependencies go?** The transport calls the domain logic, and the domain logic calls the storage. Nothing points back.
   Shared code holds nothing about one domain. Search the shared code before you plan a new helper.
10. **What happens on delete?** Prefer a delete that a person can undo. A child that only exists with its parent follows the parent.
    Derive that state from the parent. Do not write the parent's state onto each child: that cannot be undone.

## More than one customer

Answer these when the product keeps the data of more than one customer.

11. **Where is the customer boundary enforced?** Enforce it two times: in the query, and in the data store itself. A request with no customer fails loudly. It never returns all rows, and never zero rows.
    The test signs in as two customers, and the second one gets "not found" for the first one's resource.
12. **Who can reach across the boundary?** No unnamed bypass. Each reach across it says its kind (a machine job, a support person, system work inside a person's request) and its reason.
    A person's reach across it is recorded before it is granted. When the record cannot be written, the grant is refused.
13. **What does each route need?** Each route declares the permission that it needs. One test walks the real route table and fails for a route with no declaration.
14. **Where do plans and limits live?** In one place. All other code reads a limit or a feature by its key, never by the name of a plan.
15. **Which writes need a second look?** A risky write is saved as a proposal with its full payload. An approval runs it exactly one time: two approvals at the same moment cannot both run it.

## Tools for agents

16. **What can each tool do?** Each tool that an agent can call declares its risk: it reads, it writes, or it is dangerous. A tool with no declaration is refused.
    An act that belongs to the person, such as a signature or a send, is not a tool.

## Guards

Each answer above is a rule. Tenet 9 says that a rule that matters has a guard. For each rule that the plan adds, add the task that writes its guard.
The best guard is a plain test that reads the repository itself: the route table, the migrations, the build files, the production settings.

## Questions with no default

This lens gives no answer to these. Ask the person, and write the answer as a ruling in the spec:

- How does the API change without breaking a client that is already in use?
- What happens when a caller sends the same write two times?
- What happens when two callers change the same record at the same moment?
