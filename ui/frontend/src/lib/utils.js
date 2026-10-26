export function trim(value, size) {
    if (value.length > size) {
        return value.substring(0, size).concat(" ...")
    }
    return value
}

export function isNullOrEmptyObject(obj) {
    if(obj === null) {
        return true
    }
    return Object.keys(obj).length === 0;
}

export const allTables = ['inspections', 'inspection_items', 'schedules', 'templates', 'template_permissions',
    'sites', 'site_members', 'groups', 'group_users', 'schedule_assignees', 'schedule_occurrences', 'actions',
    'action_assignees', 'action_timeline_items', 'issues', 'issue_timeline_items', 'assets', 'users', 'issue_assignees',
    'training_course_progresses',
    'new_schedules', 'new_schedule_assignees', 'new_schedule_occurrences', 'action_fields', 'issue_relations',
    'investigations', 'investigation_fields', 'investigation_detail_fields', 'investigation_relationships',
    'contractor_companies', 'contractor_company_documents', 'contractor_company_members',
    'maintenance_programs', 'maintenance_plans', 'assets_maintenance', 'credential_types', 'credentials',
    'lone_worker_jobs'];
