---
references:
    - specs/apsis-lmstudio-logs/requirements.md
    - specs/apsis-lmstudio-logs/design.md
    - specs/apsis-lmstudio-logs/decision_log.md
---
# Apsis LM Studio server logs

## Phase 1: Shared representation and time zone (transcript)

- [x] 1. Add FormatLMStudio, new Entry fields and LMStudioRequest to the shared representation

- [x] 2. Implement resolveZone with the embedded tzdata and a table test

## Phase 2: Record parser and detection (transcript)

- [x] 3. Write synthetic fixtures and PROVENANCE.md under testdata/lmstudio/

- [x] 4. Implement ProbeLMStudio and wire it into DetectFormat ahead of the exec probe

- [x] 5. Implement ParseLMStudioRecords: header/record lexing and the five-row classification table

- [x] 6. Implement request/message translation: endpoint shapes, client_metadata and Parameters retention

- [x] 7. Implement ParseLMStudio and annotateExtensions with the extension-prefix property test

## Phase 3: Rendering of new content (transcript)

- [x] 8. Render LM Studio Markdown: request boundaries, unattributed events and the Parameters block

- [x] 9. Render LM Studio HTML and metadata with golden-file parity tests

## Phase 4: Shared JSON projection (sharedjson)

- [x] 10. Project requests[] and the new entry fields into session documents

- [x] 11. Add listing request_count and unattributed_summary with schema conformance tests

## Phase 5: Discovery, identity and cache (sessions)

- [x] 12. Implement the build-tagged fingerprintFile and per-file summary cache

- [x] 13. Implement root resolution, file discovery and buildLMStudioIndex

- [x] 14. Implement request/thread identifiers with the stability property test

- [x] 15. Wire SourceLMStudio into types.go and lister.go with the project-scope read-nothing guarantee

## Phase 6: Session assembly and resolution (sessions)

- [x] 16. Implement resolveLMStudio with the three identifier forms and the singleton fast path

- [x] 17. Wire PreParsed through resolver.go, parse.go and the ResolvePath rejection

## Phase 7: CLI surface and apsis serve

- [x] 18. Add the lmstudio CLI flags, agentToFormat, rejections and lmStudioInput to main.go

- [x] 19. Add the apsisweb Config, discovery-disclosure warning and source badge CSS
