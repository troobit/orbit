---
references:
    - specs/apsis-lmstudio-logs/requirements.md
    - specs/apsis-lmstudio-logs/design.md
    - specs/apsis-lmstudio-logs/decision_log.md
---
# Apsis LM Studio server logs

## Phase 1: Shared representation and time zone (transcript)

- [ ] 1. Add FormatLMStudio, new Entry fields and LMStudioRequest to the shared representation

- [ ] 2. Implement resolveZone with the embedded tzdata and a table test

## Phase 2: Record parser and detection (transcript)

- [ ] 3. Write synthetic fixtures and PROVENANCE.md under testdata/lmstudio/

- [ ] 4. Implement ProbeLMStudio and wire it into DetectFormat ahead of the exec probe

- [ ] 5. Implement ParseLMStudioRecords: header/record lexing and the five-row classification table

- [ ] 6. Implement request/message translation: endpoint shapes, client_metadata and Parameters retention

- [ ] 7. Implement ParseLMStudio and annotateExtensions with the extension-prefix property test

## Phase 3: Rendering of new content (transcript)

- [ ] 8. Render LM Studio Markdown: request boundaries, unattributed events and the Parameters block

- [ ] 9. Render LM Studio HTML and metadata with golden-file parity tests

## Phase 4: Shared JSON projection (sharedjson)

- [ ] 10. Project requests[] and the new entry fields into session documents

- [ ] 11. Add listing request_count and unattributed_summary with schema conformance tests

## Phase 5: Discovery, identity and cache (sessions)

- [ ] 12. Implement the build-tagged lmFingerprint and per-file summary cache

- [ ] 13. Implement root resolution, file discovery and buildLMStudioIndex

- [ ] 14. Implement request/thread identifiers with the stability property test

- [ ] 15. Wire SourceLMStudio into types.go and lister.go with the project-scope read-nothing guarantee

## Phase 6: Session assembly and resolution (sessions)

- [ ] 16. Implement resolveLMStudio with the three identifier forms and the singleton fast path

- [ ] 17. Wire PreParsed through resolver.go, parse.go and the ResolvePath rejection

## Phase 7: CLI surface and apsis serve

- [ ] 18. Add the lmstudio CLI flags, agentToFormat, rejections and lmStudioInput to main.go

- [ ] 19. Add the apsisweb Config, discovery-disclosure warning and source badge CSS
