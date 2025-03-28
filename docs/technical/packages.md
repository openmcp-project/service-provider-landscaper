# Package Dependencies

```mermaid
flowchart TD

    subgraph controller
        
    end

    subgraph installer
        instance --> landscaper --> resources
        instance --> helmdeployer --> resources
        instance --> manifestdeployer --> resources
        instance --> rbac --> resources
     end

    subgraph shared
        cluster
        identity
        providerconfig
        readiness
        types
    end
    
    main --> app --> controller
    controller --> installer
    installer --> shared

```