---
apiVersion: warp/v1alpha1
kind: Workspace
metadata:
  name: tasksmith
  description: TaskSmith workspace configuration.
spec:
  projects: ["."]
  defaultProvider: ollama
---
