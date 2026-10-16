<script>
    import './common.css';
    import {shadowConfig} from "../lib/store.js";
    import {push} from "svelte-spa-router";
    import Button from "../components/Button.svelte";
    import StatusBar from "../components/StatusBar.svelte";


    let data = [
        {
            "left":  { "id": "inspections", "name": "Inspections"},
            "right": { "id": "inspection_items", "name": "Inspection Items"}
        },
        {
            "left":  { "id": "templates", "name": "Templates"},
            "right": { "id": "template_permissions", "name": "Template Permissions"}
        },
        {
            "left":  { "id": "sites", "name": "Sites"},
            "right": { "id": "site_members", "name": "Site Members"}
        },
        {
            "left":  { "id": "groups", "name": "Groups"},
            "right": { "id": "group_users", "name": "Group Users"}
        },
        {
            "left":  { "id": "users", "name": "Users"},
            "right": { "id": "training_course_progresses", "name": "Training course completions"}
        },
        {
            "left":  { "id": "actions", "name": "Actions"},
            "right": { "id": "action_fields", "name": "Action Fields"}
        },
        {
            "left":  { "id": "action_assignees", "name": "Action Assignees"},
            "right": { "id": "action_timeline_items", "name": "Action Timeline Items"}
        },
        {
            "left":  { "id": "issues", "name": "Issues"},
            "right": { "id": "issue_relations", "name": "Issue Relations"}
        },
        {
            "left":  { "id": "issue_assignees", "name": "Issue Assignees"},
            "right": { "id": "issue_timeline_items", "name": "Issue Timeline Items"}
        },
        {
            "left":  { "id": "schedules", "name": "Schedules"},
            "right": { "id": "schedules_v2", "name": "Schedules (new)"}
        },
        {
            "left":  { "id": "schedule_assignees", "name": "Schedule Assignees"},
            "right": { "id": "schedule_assignees_v2", "name": "Schedule Assignees (new)"}
        },
        {
            "left":  { "id": "schedule_occurrences", "name": "Schedule Occurrences"},
            "right": { "id": "schedule_occurrences_v2", "name": "Schedule Occurrences (new)"}
        },
        {
            "left":  { "id": "investigations", "name": "Investigations"},
            "right": { "id": "investigation_fields", "name": "Investigation Fields"}
        },
        {
            "left":  { "id": "investigation_detail_fields", "name": "Investigation Detail Fields"},
            "right": { "id": "investigation_relationships", "name": "Investigation Relationships"}
        },
        {
            "left":  { "id": "assets", "name": "Assets"},
            "right": { "id": "assets_maintenance", "name": "Assets Maintenance"}
        },
        {
            "left":  { "id": "maintenance_programs", "name": "Maintenance Programs"},
            "right": { "id": "maintenance_plans", "name": "Maintenance Plans"}
        },
        {
            "left":  { "id": "contractor_companies", "name": "Contractor Companies"},
            "right": { "id": "contractor_company_documents", "name": "Contractor Company Documents"}
        },
        {
            "left":  { "id": "contractor_company_members", "name": "Contractor Company Members"},
            "right": null
        },
        {
            "left":  { "id": "credential_types", "name": "Credential Types"},
            "right": { "id": "credentials", "name": "Credentials"}
        },
        {
            "left":  { "id": "lone_worker_jobs", "name": "Lone Worker Jobs"},
            "right": null
        }
    ]

    function trim(org) {
        if (org.length > 80) {
            return org.substring(0, 80).concat(" ...")
        }
        return org
    }

    let isChecked = false;
    if($shadowConfig["Export"]["Tables"].length === 0) {
        let all = []
        data.forEach(function (e) {
            if (e.left !== null) {
                all.push(e.left.id)
            }
            if (e.right !== null) {
                all.push(e.right.id)
            }
        });
        $shadowConfig["Export"]["Tables"] = all
        isChecked = true;
    }

    function toggleHeaderCheckbox() {
        if (isChecked) {
            isChecked = false;
        }
    }

    function toggleBodyCheckboxes() {
        const checkboxes = document.querySelectorAll('.table-body input[type="checkbox"]');
        for (const checkbox of checkboxes) {
            checkbox.checked = !isChecked;
        }
    }

    function handleDone() {
        let selectedTables = [];

        const checkboxes = document.querySelectorAll('.table-body input[type="checkbox"]');
        for (const checkbox of checkboxes) {
            if (checkbox.checked) {
                selectedTables.push(checkbox.__value)
            }
        }

        let maxData = 0
        data.forEach(function (e) {
            if (e.left !== null) {
                maxData++
            }
            if (e.right !== null) {
                maxData++
            }
        });

        if (selectedTables.length === maxData) {
            $shadowConfig["Export"]["Tables"] = []
        } else {
            $shadowConfig["Export"]["Tables"] = selectedTables
        }

        push("/config")
    }
</script>

<div class="table-filter-page">
    <section class="top-nav">
        <div class="nav-left">
            <div class="h1">Data set selection</div>
        </div>
        <div class="nav-right">
            <Button label="Done" type="active-white" onClick={handleDone}/>
        </div>
    </section>

    <section class="m-top-16 table-list">
        <div class="table-header text-gray-2">
            <div class="table-row p-horiz-8">
                <input type="checkbox" class="checkbox-purple" on:click="{toggleBodyCheckboxes}" bind:checked={isChecked}/>
                <div class="m-left-32">Data set table</div>
            </div>
        </div>
        <div class="table-body text-gray-2 m-top-8">
        {#each data as { left, right }, i}
            <div class="table-row p-horiz-8">
                {#if left}
                <div class="table-cell">
                    <input type="checkbox" class="checkbox-purple" on:click={toggleHeaderCheckbox} bind:group={$shadowConfig["Export"]["Tables"]} value="{left.id}"/>
                    <img class="m-left-32" src="../images/template-icon.svg" alt="template"/>
                    <div class="m-left-8">{trim(left.name)}</div>
                </div>
                {/if}

                {#if right}
                <div class="table-cell">
                    <input type="checkbox" class="checkbox-purple" on:click={toggleHeaderCheckbox} bind:group={$shadowConfig["Export"]["Tables"]} value="{right.id}"/>
                    <img class="m-left-32" src="../images/template-icon.svg" alt="template"/>
                    <div class="m-left-8">{trim(right.name)}</div>
                </div>
                {/if}
            </div>
        {/each}
        </div>
    </section>
</div>

<StatusBar/>

<style>
    /* the status bar is fixed to the bottom of the window, so keep the page above it */
    .table-filter-page {
        --status-bar-height: 56px;
        height: calc(100vh - var(--status-bar-height));
        display: flex;
        flex-direction: column;
        padding-top: var(--main-gutter-top);
        padding-left: var(--main-gutter-left);
        padding-right: var(--main-gutter-right);
    }

    .table-list {
        flex: 1;
        min-height: 0;
        display: flex;
        flex-direction: column;
    }

    .table-body {
        /* the list is longer than the window, so scroll it in place with a visible scrollbar */
        flex: 1;
        min-height: 0;
        overflow-y: auto;
        scrollbar-width: thin;
    }

    .table-body::-webkit-scrollbar {
        width: 8px;
    }

    .table-body::-webkit-scrollbar-thumb {
        background-color: #B5BBCB;
        border-radius: 4px;
    }

    .table-header {
        background-color: #DBDFEB;
    }

    .table-header > .table-row {
        height: 36px;
        display: flex;
        align-items: center;
    }

    .table-body > .table-row {
        height: 52px;
    }

    .table-body > .table-row {
        width: 100%;
        display: flex;
    }

    .table-row > .table-cell {
        width: 50%;
        display: flex;
        align-items: center;
    }
</style>
