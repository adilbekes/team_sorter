# Team Sorter Spec v1

## Input
- Number of teams to create
- List of participants
- Each participant includes:
    - Name
    - Rating

## Validation
- Number of teams must be > 0
- Participants must not be empty
- Number of teams must be <= number of participants
- Each participant:
    - Name must not be empty
    - Rating must be >= 0
- Duplicate participant names are invalid

## Calculation
- All participants must be assigned exactly once
- Teams are created according to the requested number of teams
- Participants are distributed to make teams as balanced as possible by total rating
- Team sizes should be as even as possible
- Team size difference must not exceed 1
- Sorting must be deterministic for the same input
- Each team includes:
    - Team name
    - Total rating
    - Assigned participants
- Response meta includes:
    - Team count
    - Participant count
    - Minimum team rating
    - Maximum team rating
    - Rating difference between strongest and weakest teams

### Balancing rules
- Higher-rated participants should be distributed across teams as evenly as possible
- The algorithm should minimize the difference between team total ratings
- If multiple teams are equally suitable for assignment, the sorter should prefer:
    - Team with lower total rating
    - Then team with fewer participants
    - Then lower team index

## Errors
- ErrInvalidTeamCount
- ErrInvalidParticipants
- ErrDuplicateParticipantName
- ErrNoSolution