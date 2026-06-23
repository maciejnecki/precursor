# Objective 
Personal single user cross-platform app for tracking project progress in finish-to-start way as well as capturing design intent decisions. 

## Core Workflow 
1. User creates a project giving it a name, short description, colour, icon. 
2. Opening the project from the sidebar opens up a canvas of task nodes (panel on top 70% of the ui) with a text input field in the lower 30% of the ui 
3. Typing in the text area involves a title, longer format markdown description, icon. Colours of tasks will relate to their status. 
4. When no task node is selected, typing in the text area and saving will create a new node on the canvas. 
5. If a node is selected, typing in the text area creates a dependency that links as a precursor to the selected node. 
6. The nodes that are created not as precursors are the endpoint nodes and they get placed radially in the centre of the canvas. 
7. Then precursor nodes get visually linked and placed radially outwards from the nodes they relate to. 
8. When a node is selected, by default the text editor has a tab on the top "New Precursor", this tab can be changed to "New Decision". 
9. While standard nodes link in a final-through-child-to-initial work item manner, Decision nodes link in the opposite way as they showcase the transition of the precursor node to its parent. 
10. Multiple decision nodes can be added sequentially between the precursor and its parent.
11. Each node has a status that can be assigned to it: "Scheduled", "In Progress", "Done", "Redundant". 
12. Status transition is also done through a Decision node. 

## Additional Features
1. A project should be easy to backup in some form of open access format 
2. There should be an export feature that allows the user to create a table of all completed nodes and the decisions that have lead to their completion. 
