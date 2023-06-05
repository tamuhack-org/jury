import { useEffect, useState } from 'react';
import AdminStat from '../AdminStat';

interface ProjectStats {
    num: number;
    avg_votes: number;
    avg_seen: number;
}

const AddProjectsStatsPanel = () => {
    const [stats, setStats] = useState<ProjectStats>({ num: 0, avg_votes: 0, avg_seen: 0 });
    useEffect(() => {
        const fetchStats = async () => {
            const fetchedStats = await fetch(`${process.env.REACT_APP_JURY_URL}/project/stats`, {
                method: 'GET',
                headers: { 'Content-Type': 'application/json' },
                credentials: 'include',
            }).then((data) => data.json());
            // TODO: Add error checking
            setStats(fetchedStats);
        };

        fetchStats();
    }, []);

    return (
        <div className="flex flex-col justify-evenly w-full mt-8">
            <div className="flex justify-evenly basis-2/5">
                <AdminStat name="Active Projects" value={stats.num} />
                <AdminStat name="Average Votes" value={stats.avg_votes} />
                <AdminStat name="Average Seen" value={stats.avg_seen} />
            </div>
        </div>
    );
};

export default AddProjectsStatsPanel;
