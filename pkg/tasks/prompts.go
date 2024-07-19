package tasks

const intentPromptTemplate = `
Identify the intent of the subject's statement or question below.

If you can't derive an Intent then simply respond back with Intent: None.

EXAMPLE
Human: Does Nike make running shoes?
Assistant: The subject is inquiring about whether Nike, a specific brand, manufactures running shoes.

Human: {{.Input}}
`

type IntentPromptTemplateData struct {
	Input string
}

const defaultSummaryPromptTemplateAnthropic = `
Review the Current Summary inside <current_summary></current_summary> XML tags, 
and the New Lines of the provided conversation inside the <new_lines></new_lines> XML tags. Create a concise summary 
of the conversation, adding from the <new_lines> to the <current_summary>.
If the New Lines are meaningless or empty, return the <current_summary>.

Here is an example:
<example>
<current_summary>
The human inquires about Led Zeppelin's lead singer and other band members. The AI identifies Robert Plant as the 
lead singer.
<current_summary>
<new_lines>
Human: Who were the other members of Led Zeppelin?
Assistant: The other founding members of Led Zeppelin were Jimmy Page (guitar), John Paul Jones (bass, keyboards), and 
John Bonham (drums).
</new_lines> 
Assistant: The human inquires about Led Zeppelin's lead singer and other band members. The AI identifies Robert Plant as the lead
singer and lists the founding members as Jimmy Page, John Paul Jones, and John Bonham.
</example>

<current_summary>
{{.PrevSummary}}
</current_summary>
<new_lines>
{{.MessagesJoined}}
</new_lines>

Provide a response immediately without preamble.
`

const defaultSummaryPromptTemplateOpenAI = `
Review the Current Content, if there is one, and the New Lines of the provided conversation. Create a concise summary 
of the conversation, adding from the New Lines to the Current summary.
If the New Lines are meaningless, return the Current Content.

EXAMPLE
Current summary:
The human inquires about Led Zeppelin's lead singer and other band members. The AI identifies Robert Plant as the 
lead singer.
New lines of conversation:
Human: Who were the other members of Led Zeppelin?
AI: The other founding members of Led Zeppelin were Jimmy Page (guitar), John Paul Jones (bass, keyboards), and 
John Bonham (drums).
New summary:
The human inquires about Led Zeppelin's lead singer and other band members. The AI identifies Robert Plant as the lead
singer and lists the founding members as Jimmy Page, John Paul Jones, and John Bonham.
EXAMPLE END

Current summary:
{{.PrevSummary}}
New lines of conversation:
{{.MessagesJoined}}
New summary:
`

type SummaryPromptTemplateData struct {
	PrevSummary    string
	MessagesJoined string
}

// const defaultFactPromptTemplate = `
// Provided the messages sent by the human, identify a fact that is true and verifiable.
// You will also be provided some existing facts extracted from previous conversations if any. Review these facts and edit
// them as needed to ensure they are accurate and relevant to the current conversation. Do not generate any duplicate facts.
// Write the facts in the format "Fact: [fact]\nFact: [fact]...etc..." separated by a new line for each
// additional fact. If there are no new facts to add, or no facts to edit, or if
// the conversation is meaningless, return "Fact: None". If there are any
// contradicting facts, you may try to resolve them, or remove the contradicting
// facts. 
//
// For all existing and unchanged facts, ensure they are included in the new facts.
// Feel free to generate many facts as you need as long as they are relevant.
//
// EXAMPLE
// User Messages:
// Human: I'm thinking about travelling to Paris next summer with three friends.
// Human: I recently gave Led Zeppelin's first album a listen, turns out it's pretty good.
// Human: I love running when its cloudy outside.
// Human: I dont really like the Beatles.
//
// Existing Facts:
// Fact: The user is a fan of Queen.
// Fact: The user has no plans to listen to Led Zeppelin's first album.
// Fact: The user frequently goes bouldering at the local gym.
// Fact: The user has no current plans to travel.
//
// New Facts: 
// Fact: The user is a fan of Queen and dislikes the Beatles.
// Fact: The user recently listened to Led Zepplin's first album.
// Fact: The user frequently goes bouldering at the local gym.
// Fact: The user is considering travelling to Paris next summer with three friends.
// Fact: The user enjoys running when it's cloudy outside.
//
// EXAMPLE END
//
// Existing Facts: 
// {{.PrevFactsJoined}}
//
// User Messages:
// {{.MessagesJoined}}
// `

const defaultFactPromptTemplate = `
System Instructions:

Extract Facts and Preferences: Analyze the provided messages from the human to extract
granular facts and preferences (likes, dislikes, interests, and intentions)
that are true and verifiable.

Review Existing Facts and Preferences: Review the existing facts and
preferences from previous conversations. Edit them if necessary to ensure
accuracy and relevance to the current conversation. Avoid generating
duplicate preferences.

Resolve Contradictions: If there are any contradicting facts or preferences,
attempt to resolve them or remove the contradicting preferences altogether.
Format Preferences: Write each facts or preference on a new line in the format
"Fact: [data]". Ensure that each fact is atomic (i.e., represents a single,
indivisible preference).

Handle Edge Cases: If there are no new facts or preferences to add, no
preferences to edit, or if the conversation is meaningless, return the previous facts and preferences.
If there are no existing facts or preferences, return "Fact: None". 

Retain Unchanged Preferences: Ensure that all unchanged
facts and preferences from previous conversations are included in the new preferences
output. 

Example:
User Messages:
Human: I'm thinking about traveling to Paris next summer with three friends.
Human: I recently gave Led Zeppelin's first album a listen; turns out it's pretty good.
Human: I love running when it's cloudy outside.
Human: I don't really like the Beatles.

Existing Facts and Preferences:
Fact: The user is a fan of Queen.
Fact: The user has no plans to listen to Led Zeppelin's first album.
Fact: The user frequently goes bouldering at the local gym.
Fact: The user has no current plans to travel.

New Facts and Preferences:
Fact: The user is considering traveling to Paris next summer.
Fact: The user plans to travel to Paris with three friends.
Fact: The user recently listened to Led Zeppelin's first album.
Fact: The user likes Led Zeppelin's first album.
Fact: The user loves running when it's cloudy outside.
Fact: The user dislikes the Beatles.

End Example

Existing Facts: 
{{.PrevFactsJoined}}

User Messages:
{{.MessagesJoined}}
`

type FactPromptTemplateData struct {
	PrevFactsJoined  string
	MessagesJoined   string
}
