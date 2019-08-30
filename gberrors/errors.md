# Error Reporting

## Formatting

Don't put any specific assumptions about formatting into the message texts. Expect clients and the server log to wrap lines to fit their own needs. In long messages, newline characters (\n) can be used to indicate suggested paragraph breaks. Don't end a message with a newline. Don't use tabs or other formatting characters.

## Grammar and Punctuation

Do not capitalize the first letter. Do not end a message with any punctuation.

Use the active voice. Use complete sentences when there is an acting subject ("A could not do B"). Use telegram style without subject if the subject would be the program itself; do not use "I" for the program.

Use past tense if an attempt to do something failed, but could perhaps succeed next time (perhaps after fixing some problem). Use present tense if the failure is certainly permanent.

Spell out words in full.

## Reasons for Errors

Messages should always state the reason why an error occurred (unless it is obviously implied), and should be included in parenthesis:

`could not open file something.txt (file is in use)`

Don't include the name of the reporting routine in the error text, or blame internal components such as a database or message broker. A reason should only be included if it is potentially useful for a client to modify their request parameters.

## Examples

`account not found for id 5328c91e-94b9-4851-bb0e-aa7b17ec3271`

`failed to create account (password too short)`

`failed to place order (qty must be > 0)`
