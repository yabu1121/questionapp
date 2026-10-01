start transaction;

-- Languages
insert into language (code, name, native_name, description)
values
    ('ja', 'Japanese', '日本語', 'Japanese language'),
    ('en', 'English', 'English', 'English language'),
    ('ko', 'Korean', '한국어', 'Korean language')
on duplicate key update
    name = values(name),
    native_name = values(native_name),
    description = values(description);

-- The passwords are placeholders because these users are for GET development.
insert into users (
    id, name, display_name, handle, email, hashed_password,
    bio, avatar_url, birthday, created_at
)
values
    (1001, 'seed-taro', 'タロウ', 'seed_taro', 'seed-taro@example.com',
     'development-seed-password', 'Goとコーヒーが好きです',
     'https://cdn.example.com/avatars/seed-taro.png', '1998-04-12', '2026-01-10 09:00:00.000000'),
    (1002, 'seed-hanako', 'ハナコ', 'seed_hanako', 'seed-hanako@example.com',
     'development-seed-password', '新しい技術を試すのが好きです',
     'https://cdn.example.com/avatars/seed-hanako.png', '2000-08-25', '2026-01-11 10:00:00.000000'),
    (1003, 'seed-jiro', 'ジロウ', 'seed_jiro', 'seed-jiro@example.com',
     'development-seed-password', null,
     'https://cdn.example.com/avatars/seed-jiro.png', null, '2026-01-12 11:00:00.000000')
on duplicate key update
    name = values(name), display_name = values(display_name), bio = values(bio),
    avatar_url = values(avatar_url), birthday = values(birthday);

insert into user_setting (user_id, ui_mode, timezone)
values
    (1001, 'dark', 'Asia/Tokyo'),
    (1002, 'light', 'Asia/Tokyo'),
    (1003, 'system', 'America/New_York')
on duplicate key update
    ui_mode = values(ui_mode), timezone = values(timezone);

insert into user_notification_setting (user_id, channel, is_enabled)
values
    (1001, 'email', true),
    (1001, 'slack', false),
    (1002, 'email', false),
    (1002, 'slack', true),
    (1003, 'email', true)
on duplicate key update
    is_enabled = values(is_enabled);

insert into follows (follower_id, following_id, created_at)
values
    (1001, 1002, '2026-02-01 09:00:00.000000'),
    (1001, 1003, '2026-02-02 09:00:00.000000'),
    (1002, 1001, '2026-02-03 09:00:00.000000'),
    (1003, 1002, '2026-02-04 09:00:00.000000')
on duplicate key update
    created_at = values(created_at);

-- Different statuses, types, languages, and result visibility rules.
insert into questionnaire (
    id, created_by, title, description, status, deadline,
    type, max_choices, language_code, language_detected, visibility, created_at
)
values
    (2001, 1001, '好きなプログラミング言語',
     '一番好きな言語を1つ選んでください', 'published',
     '2030-12-31 23:59:59.000000', 'single_choice', 1, 'ja', false,
     'always', '2026-03-01 10:00:00.000000'),
    (2002, 1002, 'Tools used for development',
     'Choose up to two tools you use regularly', 'published',
     '2030-11-30 23:59:59.000000', 'multi_choices', 2, 'en', true,
     'after_vote', '2026-03-02 11:00:00.000000'),
    (2003, 1003, '朝型と夜型のどちらですか',
     '生活リズムについて教えてください', 'closed',
     '2026-06-30 23:59:59.000000', 'single_choice', 1, 'ja', false,
     'closed', '2026-03-03 12:00:00.000000'),
    (2004, 1001, '下書きアンケート',
     '作成途中のアンケートです', 'draft',
     '2030-10-31 23:59:59.000000', 'single_choice', 1, 'ja', false,
     'always', '2026-03-04 13:00:00.000000')
on duplicate key update
    created_by = values(created_by), title = values(title),
    description = values(description), status = values(status),
    deadline = values(deadline), type = values(type), max_choices = values(max_choices),
    language_code = values(language_code), language_detected = values(language_detected),
    visibility = values(visibility), created_at = values(created_at);

insert into choice (id, questionnaire_id, title, display_order)
values
    (3001, 2001, 'Go', 1),
    (3002, 2001, 'TypeScript', 2),
    (3003, 2001, 'Python', 3),
    (3004, 2002, 'Neovim', 1),
    (3005, 2002, 'Docker', 2),
    (3006, 2002, 'GitHub', 3),
    (3007, 2003, '朝型', 1),
    (3008, 2003, '夜型', 2),
    (3009, 2004, '選択肢A', 1),
    (3010, 2004, '選択肢B', 2)
on duplicate key update
    questionnaire_id = values(questionnaire_id), title = values(title),
    display_order = values(display_order);

insert into vote (id, questionnaire_id, user_id, created_at)
values
    (4001, 2001, 1002, '2026-04-01 09:00:00.000000'),
    (4002, 2001, 1003, '2026-04-01 10:00:00.000000'),
    (4003, 2002, 1001, '2026-04-02 09:00:00.000000'),
    (4004, 2003, 1001, '2026-04-03 09:00:00.000000'),
    (4005, 2003, 1002, '2026-04-03 10:00:00.000000')
on duplicate key update
    questionnaire_id = values(questionnaire_id), user_id = values(user_id),
    created_at = values(created_at);

insert into answer_item (id, vote_id, choice_id)
values
    (5001, 4001, 3001),
    (5002, 4002, 3002),
    (5003, 4003, 3004),
    (5004, 4003, 3005),
    (5005, 4004, 3008),
    (5006, 4005, 3007)
on duplicate key update
    vote_id = values(vote_id), choice_id = values(choice_id);

-- Root comments are inserted before replies to satisfy the parent foreign key.
insert into comments (id, questionnaire_id, user_id, content, parent_comment_id, created_at)
values
    (6001, 2001, 1002, 'Goに投票しました', null, '2026-05-01 09:00:00.000000'),
    (6002, 2002, 1003, 'Dockerは毎日使っています', null, '2026-05-02 09:00:00.000000'),
    (6003, 2003, 1001, '最近は夜型です', null, '2026-05-03 09:00:00.000000')
on duplicate key update
    questionnaire_id = values(questionnaire_id), user_id = values(user_id),
    content = values(content), parent_comment_id = values(parent_comment_id),
    created_at = values(created_at);

insert into comments (id, questionnaire_id, user_id, content, parent_comment_id, created_at)
values
    (6004, 2001, 1001, '投票ありがとうございます', 6001, '2026-05-01 10:00:00.000000'),
    (6005, 2002, 1002, '私もDockerを使っています', 6002, '2026-05-02 10:00:00.000000')
on duplicate key update
    questionnaire_id = values(questionnaire_id), user_id = values(user_id),
    content = values(content), parent_comment_id = values(parent_comment_id),
    created_at = values(created_at);

insert into questionnaire_like (questionnaire_id, user_id, created_at)
values
    (2001, 1001, '2026-06-01 09:00:00.000000'),
    (2001, 1002, '2026-06-01 10:00:00.000000'),
    (2001, 1003, '2026-06-01 11:00:00.000000'),
    (2002, 1001, '2026-06-02 09:00:00.000000'),
    (2003, 1002, '2026-06-03 09:00:00.000000')
on duplicate key update
    created_at = values(created_at);

commit;
