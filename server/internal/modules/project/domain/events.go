package domain

// EventArchived is the event type of a project's archive (M7 P3c spec 2.6,
// R3), published in its transaction after archived_at is written: the
// project, its workspace, the account that archived it and the moment
// written, with no change; an unarchive publishes none. The string is also
// the code of ErrArchived, the 409 of a write on an archived project; the
// two are unrelated names in separate namespaces (an event type, a problem
// code) that happen to match, so neither is ever written as the other.
const EventArchived = "project.archived"
